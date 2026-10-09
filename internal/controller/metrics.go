/*
Copyright 2021 The snapscheduler authors.

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

package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	snapv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	scheduleNameLabel      = "schedule_name"
	scheduleNamespaceLabel = "schedule_namespace"
	pvcNameLabel           = "pvc_name"
)

func scheduleLabels(scheduleName, scheduleNamespace, pvcName string) prometheus.Labels {
	return prometheus.Labels{
		scheduleNameLabel:      scheduleName,
		scheduleNamespaceLabel: scheduleNamespace,
		pvcNameLabel:           pvcName,
	}
}

func isSnapshotReady(snap *snapv1.VolumeSnapshot) bool {
	return snap.Status != nil && snap.Status.ReadyToUse != nil && *snap.Status.ReadyToUse
}

var (
	snapshotCurrentCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "snapscheduler_snapshot_current_count",
			Help: "Current number of VolumeSnapshots managed by a schedule for a given PVC.",
		},
		[]string{scheduleNameLabel, scheduleNamespaceLabel, pvcNameLabel},
	)
	snapshotCurrentReadyCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "snapscheduler_snapshot_current_ready_count",
			Help: "Current number of readyToUse VolumeSnapshots managed by a schedule for a given PVC.",
		},
		[]string{scheduleNameLabel, scheduleNamespaceLabel, pvcNameLabel},
	)
	snapshotCreateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "snapscheduler_snapshot_create_total",
			Help: "Cumulative number of snapshots created by a schedule for a given PVC.",
		},
		[]string{scheduleNameLabel, scheduleNamespaceLabel, pvcNameLabel},
	)
	snapshotReadyTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "snapscheduler_snapshot_ready_total",
			Help: "Cumulative number of snapshots that became readyToUse.",
		},
		[]string{scheduleNameLabel, scheduleNamespaceLabel, pvcNameLabel},
	)
	snapshotCreateErrorTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "snapscheduler_snapshot_create_error_total",
			Help: "Cumulative number of snapshot creation errors.",
		},
		[]string{scheduleNameLabel, scheduleNamespaceLabel, pvcNameLabel},
	)
)

func init() {
	metrics.Registry.MustRegister(
		snapshotCurrentCount,
		snapshotCurrentReadyCount,
		snapshotCreateTotal,
		snapshotReadyTotal,
		snapshotCreateErrorTotal,
	)
}

// updateSnapshotGauges sets the current snapshot count and ready count gauges
// for each PVC in the grouped snapshot map. It also removes gauge entries for
// PVCs that are no longer present (e.g. deleted PVCs).
func updateSnapshotGauges(scheduleName, scheduleNamespace string,
	grouped map[string][]snapv1.VolumeSnapshot, prevPVCs map[string]struct{}) {
	currentPVCs := make(map[string]struct{}, len(grouped))
	for pvcName, snaps := range grouped {
		currentPVCs[pvcName] = struct{}{}
		labels := scheduleLabels(scheduleName, scheduleNamespace, pvcName)
		snapshotCurrentCount.With(labels).Set(float64(len(snaps)))

		readyCount := 0
		for i := range snaps {
			if isSnapshotReady(&snaps[i]) {
				readyCount++
			}
		}
		snapshotCurrentReadyCount.With(labels).Set(float64(readyCount))
	}

	// Remove stale gauge entries for PVCs that disappeared
	for pvc := range prevPVCs {
		if _, exists := currentPVCs[pvc]; !exists {
			labels := scheduleLabels(scheduleName, scheduleNamespace, pvc)
			snapshotCurrentCount.Delete(labels)
			snapshotCurrentReadyCount.Delete(labels)
		}
	}

	// Update prevPVCs in-place
	for pvc := range prevPVCs {
		delete(prevPVCs, pvc)
	}
	for pvc := range currentPVCs {
		prevPVCs[pvc] = struct{}{}
	}
}

// updateReadyCounter increments snapshotReadyTotal for snapshots that have
// become readyToUse and haven't been counted yet. It also removes tracker
// entries for snapshots that no longer exist.
func updateReadyCounter(scheduleName, scheduleNamespace string,
	grouped map[string][]snapv1.VolumeSnapshot, tracker map[types.UID]struct{}) {
	liveUIDs := make(map[types.UID]struct{})

	for pvcName, snaps := range grouped {
		for i := range snaps {
			liveUIDs[snaps[i].UID] = struct{}{}
			if isSnapshotReady(&snaps[i]) {
				if _, tracked := tracker[snaps[i].UID]; !tracked {
					tracker[snaps[i].UID] = struct{}{}
					snapshotReadyTotal.With(scheduleLabels(scheduleName, scheduleNamespace, pvcName)).Inc()
				}
			}
		}
	}

	// Clean up tracker entries for deleted snapshots
	for uid := range tracker {
		if _, alive := liveUIDs[uid]; !alive {
			delete(tracker, uid)
		}
	}
}

// cleanupScheduleGauges removes all gauge entries for the given schedule.
func cleanupScheduleGauges(scheduleName, scheduleNamespace string) {
	partialLabels := prometheus.Labels{
		scheduleNameLabel:      scheduleName,
		scheduleNamespaceLabel: scheduleNamespace,
	}
	snapshotCurrentCount.DeletePartialMatch(partialLabels)
	snapshotCurrentReadyCount.DeletePartialMatch(partialLabels)
}
