package snapshotschedule

import (
	"context"
	"errors"
	"sort"
	"time"

	snapschedulerv1alpha1 "github.com/backube/snapscheduler/pkg/apis/snapscheduler/v1alpha1"
	"github.com/go-logr/logr"
	snapv1alpha1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// expireByCount deletes the oldest snapshots until the number of snapshots for
// a given PVC (created by the supplied schedule) is no more than the
// schedule's maxCount. This function is the entry point for count-based
// expiration of snapshots.
func expireByCount(schedule *snapschedulerv1alpha1.SnapshotSchedule,
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
		if len(list.Items) > int(*schedule.Spec.Retention.MaxCount) {
			list.Items = list.Items[:len(list.Items)-int(*schedule.Spec.Retention.MaxCount)]
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
func expireByTime(schedule *snapschedulerv1alpha1.SnapshotSchedule,
	logger logr.Logger, c client.Client) error {
	expiration, err := getExpirationTime(schedule, time.Now(), logger)
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
		"total", len(snapList.Items), "expired", len(expiredSnaps.Items))
	err = deleteSnapshots(expiredSnaps, logger, c)
	return err
}

func deleteSnapshots(snapshots *snapv1alpha1.VolumeSnapshotList, logger logr.Logger, c client.Client) error {
	if snapshots == nil {
		return nil
	}
	for _, snap := range snapshots.Items {
		err := c.Delete(context.TODO(), &snap, client.PropagationPolicy(metav1.DeletePropagationBackground))
		if err != nil {
			logger.Error(err, "error deleting snapshot", "name", snap.Name)
			return err
		}
	}
	return nil
}

// getExpirationTime returns the cutoff Time for snapshots created with the
// referenced schedule. Any snapshot created prior to the returned time should
// be considered expired.
func getExpirationTime(schedule *snapschedulerv1alpha1.SnapshotSchedule,
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
func filterExpiredSnaps(snaps *snapv1alpha1.VolumeSnapshotList,
	expiration time.Time) *snapv1alpha1.VolumeSnapshotList {
	outList := &snapv1alpha1.VolumeSnapshotList{}
	for _, snap := range snaps.Items {
		if snap.CreationTimestamp.Time.Before(expiration) {
			outList.Items = append(outList.Items, snap)
		}
	}
	return outList
}

// snapshotsFromSchedule returns a list of snapshots that were created by the
// supplied schedule
func snapshotsFromSchedule(schedule *snapschedulerv1alpha1.SnapshotSchedule,
	logger logr.Logger, c client.Client) (*snapv1alpha1.VolumeSnapshotList, error) {
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

	snapList := &snapv1alpha1.VolumeSnapshotList{}
	err = c.List(context.TODO(),
		&client.ListOptions{LabelSelector: selector, Namespace: schedule.Namespace}, snapList)
	if err != nil {
		logger.Error(err, "unable to retrieve list of snapshots")
		return nil, err
	}

	return snapList, nil
}

// groupSnapsByPVC takes a list of snapshots and groups them by the PVC they
// were created from
func groupSnapsByPVC(snaps *snapv1alpha1.VolumeSnapshotList) map[string]*snapv1alpha1.VolumeSnapshotList {
	groupedSnaps := make(map[string]*snapv1alpha1.VolumeSnapshotList)
	for _, snap := range snaps.Items {
		if snap.Spec.Source != nil {
			pvcName := snap.Spec.Source.Name
			if groupedSnaps[pvcName] == nil {
				groupedSnaps[pvcName] = &snapv1alpha1.VolumeSnapshotList{}
			}
			groupedSnaps[pvcName].Items = append(groupedSnaps[pvcName].Items, snap)
		}
	}

	return groupedSnaps
}

// sortSnapsByTime sorts the snapshots in order of ascending CreationTimestamp
func sortSnapsByTime(snaps *snapv1alpha1.VolumeSnapshotList) *snapv1alpha1.VolumeSnapshotList {
	if snaps == nil {
		return nil
	}
	list := snaps.DeepCopy()
	sort.Slice(list.Items, func(i, j int) bool {
		return list.Items[i].CreationTimestamp.Before(&list.Items[j].CreationTimestamp)
	})
	return list
}
