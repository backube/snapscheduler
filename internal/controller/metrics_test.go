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
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	snapv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	//nolint:revive  // Allow . import
	. "github.com/onsi/ginkgo/v2"
	//nolint:revive  // Allow . import
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

var _ = Describe("Snapshot metrics", func() {
	AfterEach(func() {
		snapshotCurrentCount.Reset()
		snapshotCurrentReadyCount.Reset()
		snapshotReadyTotal.Reset()
	})

	Describe("updateSnapshotGauges", func() {
		It("sets correct counts for snapshots per PVC", func() {
			readyTrue := true
			grouped := map[string][]snapv1.VolumeSnapshot{
				"pvc1": {
					{
						ObjectMeta: metav1.ObjectMeta{Name: "snap1"},
						Status: &snapv1.VolumeSnapshotStatus{
							ReadyToUse: &readyTrue,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{Name: "snap2"},
						Status:     nil,
					},
				},
				"pvc2": {
					{
						ObjectMeta: metav1.ObjectMeta{Name: "snap3"},
						Status: &snapv1.VolumeSnapshotStatus{
							ReadyToUse: &readyTrue,
						},
					},
				},
			}

			updateSnapshotGauges("sched1", "ns1", grouped)

			labels1 := prometheus.Labels{
				"schedule_name": "sched1", "schedule_namespace": "ns1", "pvc_name": "pvc1",
			}
			labels2 := prometheus.Labels{
				"schedule_name": "sched1", "schedule_namespace": "ns1", "pvc_name": "pvc2",
			}
			Expect(testutil.ToFloat64(snapshotCurrentCount.With(labels1))).To(Equal(float64(2)))
			Expect(testutil.ToFloat64(snapshotCurrentReadyCount.With(labels1))).To(Equal(float64(1)))
			Expect(testutil.ToFloat64(snapshotCurrentCount.With(labels2))).To(Equal(float64(1)))
			Expect(testutil.ToFloat64(snapshotCurrentReadyCount.With(labels2))).To(Equal(float64(1)))
		})

		It("handles empty grouped map", func() {
			updateSnapshotGauges("sched1", "ns1", map[string][]snapv1.VolumeSnapshot{})
			// No panic, no metrics created
		})

		It("counts zero ready when status is nil", func() {
			grouped := map[string][]snapv1.VolumeSnapshot{
				"pvc1": {
					{ObjectMeta: metav1.ObjectMeta{Name: "snap1"}, Status: nil},
				},
			}
			updateSnapshotGauges("sched1", "ns1", grouped)

			labels := prometheus.Labels{
				"schedule_name": "sched1", "schedule_namespace": "ns1", "pvc_name": "pvc1",
			}
			Expect(testutil.ToFloat64(snapshotCurrentCount.With(labels))).To(Equal(float64(1)))
			Expect(testutil.ToFloat64(snapshotCurrentReadyCount.With(labels))).To(Equal(float64(0)))
		})
	})

	Describe("updateReadyCounter", func() {
		It("increments counter for newly ready snapshots", func() {
			readyTrue := true
			tracker := make(map[types.UID]struct{})
			grouped := map[string][]snapv1.VolumeSnapshot{
				"pvc1": {
					{
						ObjectMeta: metav1.ObjectMeta{Name: "snap1", UID: "uid-1"},
						Status: &snapv1.VolumeSnapshotStatus{
							ReadyToUse: &readyTrue,
						},
					},
				},
			}

			updateReadyCounter("sched1", "ns1", grouped, tracker)

			labels := prometheus.Labels{
				"schedule_name": "sched1", "schedule_namespace": "ns1", "pvc_name": "pvc1",
			}
			Expect(testutil.ToFloat64(snapshotReadyTotal.With(labels))).To(Equal(float64(1)))
			Expect(tracker).To(HaveKey(types.UID("uid-1")))
		})

		It("does not double-count already tracked snapshots", func() {
			readyTrue := true
			tracker := map[types.UID]struct{}{
				"uid-1": {},
			}
			grouped := map[string][]snapv1.VolumeSnapshot{
				"pvc1": {
					{
						ObjectMeta: metav1.ObjectMeta{Name: "snap1", UID: "uid-1"},
						Status: &snapv1.VolumeSnapshotStatus{
							ReadyToUse: &readyTrue,
						},
					},
				},
			}

			updateReadyCounter("sched1", "ns1", grouped, tracker)

			labels := prometheus.Labels{
				"schedule_name": "sched1", "schedule_namespace": "ns1", "pvc_name": "pvc1",
			}
			// Counter should be 0 since uid-1 was already tracked
			Expect(testutil.ToFloat64(snapshotReadyTotal.With(labels))).To(Equal(float64(0)))
		})

		It("cleans up tracker entries for deleted snapshots", func() {
			tracker := map[types.UID]struct{}{
				"uid-deleted": {},
			}
			grouped := map[string][]snapv1.VolumeSnapshot{}

			updateReadyCounter("sched1", "ns1", grouped, tracker)

			Expect(tracker).NotTo(HaveKey(types.UID("uid-deleted")))
		})

		It("skips snapshots that are not ready", func() {
			readyFalse := false
			tracker := make(map[types.UID]struct{})
			grouped := map[string][]snapv1.VolumeSnapshot{
				"pvc1": {
					{
						ObjectMeta: metav1.ObjectMeta{Name: "snap1", UID: "uid-1"},
						Status: &snapv1.VolumeSnapshotStatus{
							ReadyToUse: &readyFalse,
						},
					},
				},
			}

			updateReadyCounter("sched1", "ns1", grouped, tracker)

			Expect(tracker).NotTo(HaveKey(types.UID("uid-1")))
		})
	})

	Describe("cleanupScheduleGauges", func() {
		It("removes gauge entries for a schedule", func() {
			labels := prometheus.Labels{
				"schedule_name": "sched1", "schedule_namespace": "ns1", "pvc_name": "pvc1",
			}
			snapshotCurrentCount.With(labels).Set(5)
			snapshotCurrentReadyCount.With(labels).Set(3)

			cleanupScheduleGauges("sched1", "ns1")

			// After cleanup, getting the metric should return 0 (fresh counter)
			Expect(testutil.ToFloat64(snapshotCurrentCount.With(labels))).To(Equal(float64(0)))
			Expect(testutil.ToFloat64(snapshotCurrentReadyCount.With(labels))).To(Equal(float64(0)))
		})

		It("does not affect other schedules", func() {
			labels1 := prometheus.Labels{
				"schedule_name": "sched1", "schedule_namespace": "ns1", "pvc_name": "pvc1",
			}
			labels2 := prometheus.Labels{
				"schedule_name": "sched2", "schedule_namespace": "ns1", "pvc_name": "pvc1",
			}
			snapshotCurrentCount.With(labels1).Set(5)
			snapshotCurrentCount.With(labels2).Set(10)

			cleanupScheduleGauges("sched1", "ns1")

			Expect(testutil.ToFloat64(snapshotCurrentCount.With(labels2))).To(Equal(float64(10)))
		})
	})

	Describe("snapshotReadyPredicate", func() {
		pred := snapshotReadyPredicate()

		It("ignores create events", func() {
			Expect(pred.Create(event.CreateEvent{})).To(BeFalse())
		})

		It("ignores delete events", func() {
			Expect(pred.Delete(event.DeleteEvent{})).To(BeFalse())
		})

		It("triggers on readyToUse change from false to true", func() {
			readyFalse := false
			readyTrue := true
			e := event.UpdateEvent{
				ObjectOld: &snapv1.VolumeSnapshot{
					Status: &snapv1.VolumeSnapshotStatus{ReadyToUse: &readyFalse},
				},
				ObjectNew: &snapv1.VolumeSnapshot{
					Status: &snapv1.VolumeSnapshotStatus{ReadyToUse: &readyTrue},
				},
			}
			Expect(pred.Update(e)).To(BeTrue())
		})

		It("does not trigger when readyToUse stays the same", func() {
			readyTrue := true
			e := event.UpdateEvent{
				ObjectOld: &snapv1.VolumeSnapshot{
					Status: &snapv1.VolumeSnapshotStatus{ReadyToUse: &readyTrue},
				},
				ObjectNew: &snapv1.VolumeSnapshot{
					Status: &snapv1.VolumeSnapshotStatus{ReadyToUse: &readyTrue},
				},
			}
			Expect(pred.Update(e)).To(BeFalse())
		})

		It("does not trigger for non-VolumeSnapshot objects", func() {
			e := event.UpdateEvent{
				ObjectOld: &snapv1.VolumeSnapshot{},
				ObjectNew: &snapv1.VolumeSnapshot{},
			}
			Expect(pred.Update(e)).To(BeFalse())
		})
	})

	Describe("mapSnapshotToSchedule", func() {
		It("maps a snapshot with schedule label to the correct schedule", func() {
			r := &SnapshotScheduleReconciler{}
			snap := &snapv1.VolumeSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "snap1",
					Namespace: "ns1",
					Labels: map[string]string{
						ScheduleKey: "my-schedule",
					},
				},
			}

			requests := r.mapSnapshotToSchedule(context.TODO(), snap)
			Expect(requests).To(HaveLen(1))
			Expect(requests[0].Name).To(Equal("my-schedule"))
			Expect(requests[0].Namespace).To(Equal("ns1"))
		})

		It("returns nil for snapshots without schedule label", func() {
			r := &SnapshotScheduleReconciler{}
			snap := &snapv1.VolumeSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "snap1",
					Namespace: "ns1",
				},
			}

			requests := r.mapSnapshotToSchedule(context.TODO(), snap)
			Expect(requests).To(BeNil())
		})
	})
})
