/*
Copyright (C) 2019  The snapscheduler authors

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package controllers

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/go-logr/logr"
	snapv1 "github.com/kubernetes-csi/external-snapshotter/client/v3/apis/volumesnapshot/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	snapschedulerv1 "github.com/backube/snapscheduler/api/v1"
)

// expireByCount deletes the oldest snapshots until the number of snapshots for
// a given PVC (created by the supplied schedule) is no more than the
// schedule's maxCount. This function is the entry point for count-based
// expiration of snapshots.
func expireByCount(schedule *snapschedulerv1.SnapshotSchedule,
	logger logr.Logger, c client.Client) error {
	if schedule.Spec.Retention.MaxCount == nil {
		// No count-based retention configured
		return nil
	}

	snapList, err := snapshotsFromSchedule(schedule, logger, c)
	if err != nil {
		logger.Error(err, "unable to retrieve list of snapshots")
		return err
	}

	grouped := groupSnapsByPVC(snapList)
	for _, list := range grouped {
		list = sortSnapsByTime(list)
		if len(list) > int(*schedule.Spec.Retention.MaxCount) {
			list = list[:len(list)-int(*schedule.Spec.Retention.MaxCount)]
			err := deleteSnapshots(list, logger, c)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// expireByTime deletes snapshots that are older than the retention time in the
// specified schedule. It only affects snapshots that were created by the provided schedule.
// This function is the entry point for the time-based expiration of snapshots
func expireByTime(schedule *snapschedulerv1.SnapshotSchedule, now time.Time,
	logger logr.Logger, c client.Client) error {
	expiration, err := getExpirationTime(schedule, now, logger)
	if err != nil {
		logger.Error(err, "unable to determine snapshot expiration time")
		return err
	}
	if expiration == nil {
		// No time-based retention configured
		return nil
	}

	snapList, err := snapshotsFromSchedule(schedule, logger, c)
	if err != nil {
		logger.Error(err, "unable to retrieve list of snapshots")
		return err
	}

	expiredSnaps := filterExpiredSnaps(snapList, *expiration)

	logger.Info("deleting expired snapshots", "expiration", expiration.Format(time.RFC3339),
		"total", len(snapList), "expired", len(expiredSnaps))
	err = deleteSnapshots(expiredSnaps, logger, c)
	return err
}

func deleteSnapshots(snapshots []snapv1.VolumeSnapshot, logger logr.Logger, c client.Client) error {
	for i := range snapshots {
		snap := snapshots[i]
		if err := c.Delete(context.TODO(), &snap, client.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil {
			logger.Error(err, "error deleting snapshot", "name", snap.Name)
			return err
		}
	}
	return nil
}

// getExpirationTime returns the cutoff Time for snapshots created with the
// referenced schedule. Any snapshot created prior to the returned time should
// be considered expired.
func getExpirationTime(schedule *snapschedulerv1.SnapshotSchedule,
	now time.Time, logger logr.Logger) (*time.Time, error) {
	if schedule.Spec.Retention.Expires == "" {
		// No time-based retention configured
		return nil, nil
	}

	lifetime, err := time.ParseDuration(schedule.Spec.Retention.Expires)
	if err != nil {
		logger.Error(err, "unable to parse spec.retention.expires")
		return nil, err
	}

	if lifetime < 0 {
		err := errors.New("duration must be greater than 0")
		logger.Error(err, "invalid value for spec.retention.expires")
		return nil, err
	}

	expiration := now.Add(-lifetime).UTC()
	return &expiration, nil
}

// filterExpiredSnaps returns the set of expired snapshots from the provided list.
func filterExpiredSnaps(snaps []snapv1.VolumeSnapshot,
	expiration time.Time) []snapv1.VolumeSnapshot {
	outList := make([]snapv1.VolumeSnapshot, 0)
	for _, snap := range snaps {
		if snap.CreationTimestamp.Time.Before(expiration) {
			outList = append(outList, snap)
		}
	}
	return outList
}

// snapshotsFromSchedule returns a list of snapshots that were created by the
// supplied schedule
func snapshotsFromSchedule(schedule *snapschedulerv1.SnapshotSchedule,
	logger logr.Logger, c client.Client) ([]snapv1.VolumeSnapshot, error) {
	labelSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			ScheduleKey: schedule.Name,
		},
	}
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		logger.Error(err, "unable to create label selector for snapshot expiration")
		return nil, err
	}

	listOpts := []client.ListOption{
		client.InNamespace(schedule.Namespace),
		client.MatchingLabelsSelector{
			Selector: selector,
		},
	}
	var snapList snapv1.VolumeSnapshotList
	err = c.List(context.TODO(), &snapList, listOpts...)
	if err != nil {
		logger.Error(err, "unable to retrieve list of snapshots")
		return nil, err
	}

	return snapList.Items, nil
}

// groupSnapsByPVC takes a list of snapshots and groups them by the PVC they
// were created from
func groupSnapsByPVC(snaps []snapv1.VolumeSnapshot) map[string][]snapv1.VolumeSnapshot {
	groupedSnaps := make(map[string][]snapv1.VolumeSnapshot)
	for _, snap := range snaps {
		pvcName := snap.Spec.Source.PersistentVolumeClaimName
		if pvcName != nil {
			if groupedSnaps[*pvcName] == nil {
				groupedSnaps[*pvcName] = []snapv1.VolumeSnapshot{}
			}
			groupedSnaps[*pvcName] = append(groupedSnaps[*pvcName], snap)
		}
	}

	return groupedSnaps
}

// sortSnapsByTime sorts the snapshots in order of ascending CreationTimestamp
func sortSnapsByTime(snaps []snapv1.VolumeSnapshot) []snapv1.VolumeSnapshot {
	sorted := append([]snapv1.VolumeSnapshot(nil), snaps...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].CreationTimestamp.Before(&sorted[j].CreationTimestamp)
	})
	return sorted
}
