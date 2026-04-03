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

var (
	snapshotCurrentCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "snapscheduler_snapshot_current_count",
			Help: "Current number of VolumeSnapshots managed by a schedule for a given PVC.",
		},
		[]string{"schedule_name", "schedule_namespace", "pvc_name"},
	)
	snapshotCurrentReadyCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "snapscheduler_snapshot_current_ready_count",
			Help: "Current number of readyToUse VolumeSnapshots managed by a schedule for a given PVC.",
		},
		[]string{"schedule_name", "schedule_namespace", "pvc_name"},
	)
	snapshotCreateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "snapscheduler_snapshot_create_total",
			Help: "Cumulative number of snapshots created by a schedule for a given PVC.",
		},
		[]string{"schedule_name", "schedule_namespace", "pvc_name"},
	)
	snapshotReadyTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "snapscheduler_snapshot_ready_total",
			Help: "Cumulative number of snapshots that became readyToUse.",
		},
		[]string{"schedule_name", "schedule_namespace", "pvc_name"},
	)
	snapshotCreateErrorTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "snapscheduler_snapshot_create_error_total",
			Help: "Cumulative number of snapshot creation errors.",
		},
		[]string{"schedule_name", "schedule_namespace", "pvc_name"},
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
// for each PVC in the grouped snapshot map.
func updateSnapshotGauges(scheduleName, scheduleNamespace string,
	grouped map[string][]snapv1.VolumeSnapshot) {
	for pvcName, snaps := range grouped {
		labels := prometheus.Labels{
			"schedule_name":      scheduleName,
			"schedule_namespace": scheduleNamespace,
			"pvc_name":           pvcName,
		}
		snapshotCurrentCount.With(labels).Set(float64(len(snaps)))

		readyCount := 0
		for i := range snaps {
			if snaps[i].Status != nil && snaps[i].Status.ReadyToUse != nil && *snaps[i].Status.ReadyToUse {
				readyCount++
			}
		}
		snapshotCurrentReadyCount.With(labels).Set(float64(readyCount))
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
			if snaps[i].Status != nil && snaps[i].Status.ReadyToUse != nil && *snaps[i].Status.ReadyToUse {
				if _, tracked := tracker[snaps[i].UID]; !tracked {
					tracker[snaps[i].UID] = struct{}{}
					snapshotReadyTotal.With(prometheus.Labels{
						"schedule_name":      scheduleName,
						"schedule_namespace": scheduleNamespace,
						"pvc_name":           pvcName,
					}).Inc()
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
		"schedule_name":      scheduleName,
		"schedule_namespace": scheduleNamespace,
	}
	snapshotCurrentCount.DeletePartialMatch(partialLabels)
	snapshotCurrentReadyCount.DeletePartialMatch(partialLabels)
}
