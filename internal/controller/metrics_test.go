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
	"github.com/prometheus/client_golang/prometheus/testutil"

	snapv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	//nolint:revive  // Allow . import
	. "github.com/onsi/ginkgo/v2"
	//nolint:revive  // Allow . import
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	testPVC1         = "pvc1"
	testPVC2         = "pvc2"
	testSnapshot1    = "snap1"
	testNamespace1   = "ns1"
	testSnapshotUID1 = "uid-1"
	testSchedule1    = "sched1"
)

var _ = Describe("Snapshot metrics", func() {
	AfterEach(func() {
		snapshotCurrentCount.Reset()
		snapshotCurrentReadyCount.Reset()
		snapshotReadyTotal.Reset()
	})

	It("uses stable Prometheus label names", func() {
		Expect(scheduleLabels(testSchedule1, testNamespace1, testPVC1)).To(Equal(prometheus.Labels{
			"schedule_name":      testSchedule1,
			"schedule_namespace": testNamespace1,
			"pvc_name":           testPVC1,
		}))
	})

	Describe("updateSnapshotGauges", func() {
		It("sets correct counts for snapshots per PVC", func() {
			readyTrue := true
			grouped := map[string][]snapv1.VolumeSnapshot{
				testPVC1: {
					{
						ObjectMeta: metav1.ObjectMeta{Name: testSnapshot1},
						Status: &snapv1.VolumeSnapshotStatus{
							ReadyToUse: &readyTrue,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{Name: "snap2"},
						Status:     nil,
					},
				},
				testPVC2: {
					{
						ObjectMeta: metav1.ObjectMeta{Name: "snap3"},
						Status: &snapv1.VolumeSnapshotStatus{
							ReadyToUse: &readyTrue,
						},
					},
				},
			}

			updateSnapshotGauges(testSchedule1, testNamespace1, grouped, make(map[string]struct{}))

			labels1 := prometheus.Labels{
				scheduleNameLabel: testSchedule1, scheduleNamespaceLabel: testNamespace1, pvcNameLabel: testPVC1,
			}
			labels2 := prometheus.Labels{
				scheduleNameLabel: testSchedule1, scheduleNamespaceLabel: testNamespace1, pvcNameLabel: testPVC2,
			}
			Expect(testutil.ToFloat64(snapshotCurrentCount.With(labels1))).To(Equal(float64(2)))
			Expect(testutil.ToFloat64(snapshotCurrentReadyCount.With(labels1))).To(Equal(float64(1)))
			Expect(testutil.ToFloat64(snapshotCurrentCount.With(labels2))).To(Equal(float64(1)))
			Expect(testutil.ToFloat64(snapshotCurrentReadyCount.With(labels2))).To(Equal(float64(1)))
		})

		It("handles empty grouped map", func() {
			updateSnapshotGauges(testSchedule1, testNamespace1, map[string][]snapv1.VolumeSnapshot{}, make(map[string]struct{}))
			// No panic, no metrics created
		})

		It("counts zero ready when status is nil", func() {
			grouped := map[string][]snapv1.VolumeSnapshot{
				testPVC1: {
					{ObjectMeta: metav1.ObjectMeta{Name: testSnapshot1}, Status: nil},
				},
			}
			updateSnapshotGauges(testSchedule1, testNamespace1, grouped, make(map[string]struct{}))

			labels := prometheus.Labels{
				scheduleNameLabel: testSchedule1, scheduleNamespaceLabel: testNamespace1, pvcNameLabel: testPVC1,
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
				testPVC1: {
					{
						ObjectMeta: metav1.ObjectMeta{Name: testSnapshot1, UID: testSnapshotUID1},
						Status: &snapv1.VolumeSnapshotStatus{
							ReadyToUse: &readyTrue,
						},
					},
				},
			}

			updateReadyCounter(testSchedule1, testNamespace1, grouped, tracker)

			labels := prometheus.Labels{
				scheduleNameLabel: testSchedule1, scheduleNamespaceLabel: testNamespace1, pvcNameLabel: testPVC1,
			}
			Expect(testutil.ToFloat64(snapshotReadyTotal.With(labels))).To(Equal(float64(1)))
			Expect(tracker).To(HaveKey(types.UID(testSnapshotUID1)))
		})

		It("does not double-count already tracked snapshots", func() {
			readyTrue := true
			tracker := map[types.UID]struct{}{
				testSnapshotUID1: {},
			}
			grouped := map[string][]snapv1.VolumeSnapshot{
				testPVC1: {
					{
						ObjectMeta: metav1.ObjectMeta{Name: testSnapshot1, UID: testSnapshotUID1},
						Status: &snapv1.VolumeSnapshotStatus{
							ReadyToUse: &readyTrue,
						},
					},
				},
			}

			updateReadyCounter(testSchedule1, testNamespace1, grouped, tracker)

			labels := prometheus.Labels{
				scheduleNameLabel: testSchedule1, scheduleNamespaceLabel: testNamespace1, pvcNameLabel: testPVC1,
			}
			// Counter should be 0 since the snapshot UID was already tracked
			Expect(testutil.ToFloat64(snapshotReadyTotal.With(labels))).To(Equal(float64(0)))
		})

		It("cleans up tracker entries for deleted snapshots", func() {
			tracker := map[types.UID]struct{}{
				"uid-deleted": {},
			}
			grouped := map[string][]snapv1.VolumeSnapshot{}

			updateReadyCounter(testSchedule1, testNamespace1, grouped, tracker)

			Expect(tracker).NotTo(HaveKey(types.UID("uid-deleted")))
		})

		It("skips snapshots that are not ready", func() {
			readyFalse := false
			tracker := make(map[types.UID]struct{})
			grouped := map[string][]snapv1.VolumeSnapshot{
				testPVC1: {
					{
						ObjectMeta: metav1.ObjectMeta{Name: testSnapshot1, UID: testSnapshotUID1},
						Status: &snapv1.VolumeSnapshotStatus{
							ReadyToUse: &readyFalse,
						},
					},
				},
			}

			updateReadyCounter(testSchedule1, testNamespace1, grouped, tracker)

			Expect(tracker).NotTo(HaveKey(types.UID(testSnapshotUID1)))
		})
	})

	Describe("cleanupScheduleGauges", func() {
		It("removes gauge entries for a schedule", func() {
			labels := prometheus.Labels{
				scheduleNameLabel: testSchedule1, scheduleNamespaceLabel: testNamespace1, pvcNameLabel: testPVC1,
			}
			snapshotCurrentCount.With(labels).Set(5)
			snapshotCurrentReadyCount.With(labels).Set(3)

			cleanupScheduleGauges(testSchedule1, testNamespace1)

			// After cleanup, getting the metric should return 0 (fresh counter)
			Expect(testutil.ToFloat64(snapshotCurrentCount.With(labels))).To(Equal(float64(0)))
			Expect(testutil.ToFloat64(snapshotCurrentReadyCount.With(labels))).To(Equal(float64(0)))
		})

		It("does not affect other schedules", func() {
			labels1 := prometheus.Labels{
				scheduleNameLabel: testSchedule1, scheduleNamespaceLabel: testNamespace1, pvcNameLabel: testPVC1,
			}
			labels2 := prometheus.Labels{
				scheduleNameLabel: "sched2", scheduleNamespaceLabel: testNamespace1, pvcNameLabel: testPVC1,
			}
			snapshotCurrentCount.With(labels1).Set(5)
			snapshotCurrentCount.With(labels2).Set(10)

			cleanupScheduleGauges(testSchedule1, testNamespace1)

			Expect(testutil.ToFloat64(snapshotCurrentCount.With(labels2))).To(Equal(float64(10)))
		})
	})

})
