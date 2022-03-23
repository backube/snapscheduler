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

// nolint funlen  // Long test functions ok
package controllers

import (
	"context"
	"strings"
	"time"

	snapv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	snapschedulerv1 "github.com/backube/snapscheduler/api/v1"
)

const (
	timeout  = "30s"
	interval = "100ms"
)

var logger = logf.Log

var _ = Describe("Snapshot expiration time is parsed correctly", func() {
	When("no retention time is set", func() {
		It("returns a nil expiration time", func() {
			s := &snapschedulerv1.SnapshotSchedule{}
			expiration, err := getExpirationTime(s, time.Now(), logger)
			Expect(expiration).To(BeNil())
			Expect(err).NotTo(HaveOccurred())
		})
	})
	When("the retention time is unparsable", func() {
		It("returns an error", func() {
			s := &snapschedulerv1.SnapshotSchedule{}
			s.Spec.Retention.Expires = "garbage"
			_, err := getExpirationTime(s, time.Now(), logger)
			Expect(err).To(HaveOccurred())
		})
	})
	When("the retention time is negative", func() {
		It("returns an error", func() {
			s := &snapschedulerv1.SnapshotSchedule{}
			s.Spec.Retention.Expires = "-10s"
			_, err := getExpirationTime(s, time.Now(), logger)
			Expect(err).To(HaveOccurred())
		})
	})
	When("the retention time is valid", func() {
		It("calculates the expiration time correctly", func() {
			s := &snapschedulerv1.SnapshotSchedule{}
			s.Spec.Retention.Expires = "1h"
			theTime, _ := time.Parse(timeFormat, "2013-02-01T11:04:05Z")
			expected := theTime.Add(-1 * time.Hour)
			expiration, err := getExpirationTime(s, theTime, logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(*expiration).To(Equal(expected))
		})
	})
})

var _ = Describe("Finding snapshots created by a schedule", func() {
	var objects []client.Object
	var ns *v1.Namespace
	BeforeEach(func() {
		ns = &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
			},
		}
		Expect(k8sClient.Create(context.TODO(), ns)).To(Succeed())
		Expect(ns.Name).NotTo(BeEmpty())

		objects = []client.Object{
			&snapv1.VolumeSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: ns.Name,
					Labels: map[string]string{
						"foo":       "bar",
						ScheduleKey: "s1",
					},
				},
			},
			&snapv1.VolumeSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: ns.Name,
					Labels: map[string]string{
						"foo":       "bar",
						ScheduleKey: "s1",
					},
				},
			},
			&snapv1.VolumeSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "baz",
					Namespace: ns.Name,
					Labels: map[string]string{
						"foo":       "bar",
						ScheduleKey: "s2",
					},
				},
			},
		}
	})
	AfterEach(func() {
		Expect(k8sClient.Delete(context.TODO(), ns)).To(Succeed())
	})
	JustBeforeEach(func() {
		for _, o := range objects {
			Expect(k8sClient.Create(context.TODO(), o)).To(Succeed())
			snap := &snapv1.VolumeSnapshot{}
			Eventually(func() error {
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(o), snap)
			}).Should(Succeed())
		}
	})
	When("an invalid schedule name is used", func() {
		It("should return an error", func() {
			s := &snapschedulerv1.SnapshotSchedule{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "%%!! Invalid !!%%",
					Namespace: ns.Name,
				},
			}
			_, err := snapshotsFromSchedule(context.TODO(), s, logger, k8sClient)
			Expect(err).To(HaveOccurred())
		})
	})
	Context("lookup", func() {
		It("should succeed", func() {
			s := &snapschedulerv1.SnapshotSchedule{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "s1",
					Namespace: ns.Name,
				},
			}
			snapList, err := snapshotsFromSchedule(context.TODO(), s, logger, k8sClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(snapList)).To(Equal(2))
			for _, snap := range snapList {
				Expect(snap.Name).To(Or(Equal("foo"), Equal("bar")))
			}
		})
	})
})

var _ = Describe("Expiring snapshots by time", func() {
	var ns1, ns2 *v1.Namespace
	var data []struct {
		namespace string
		schedule  string
	}
	BeforeEach(func() {
		ns1 = &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
			},
		}
		Expect(k8sClient.Create(context.TODO(), ns1)).To(Succeed())
		Expect(ns1.Name).NotTo(BeEmpty())
		ns2 = &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
			},
		}
		Expect(k8sClient.Create(context.TODO(), ns2)).To(Succeed())
		Expect(ns2.Name).NotTo(BeEmpty())

		data = []struct {
			namespace string
			schedule  string
		}{
			{ns1.Name, "schedule"},
			{ns2.Name, "schedule"},
			{ns1.Name, "different"},
			{ns2.Name, "different"},
		}
		for _, d := range data {
			snap := snapv1.VolumeSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      d.namespace + "-" + d.schedule,
					Namespace: d.namespace,
					Labels: map[string]string{
						"foo":       "bar",
						ScheduleKey: d.schedule,
					},
				},
			}
			Expect(k8sClient.Create(context.TODO(), &snap)).To(Succeed())
		}
	})
	AfterEach(func() {
		Expect(k8sClient.Delete(context.TODO(), ns1)).To(Succeed())
		Expect(k8sClient.Delete(context.TODO(), ns2)).To(Succeed())
	})
	When("a schedule doesn't have an expiration", func() {
		It("doesn't remove any snapshots", func() {
			noexpire := &snapschedulerv1.SnapshotSchedule{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "schedule",
					Namespace: ns1.Name,
				},
			}
			Expect(expireByTime(context.TODO(), noexpire, time.Now(), logger, k8sClient)).To(Succeed())

			Eventually(func() int {
				snapList := &snapv1.VolumeSnapshotList{}
				Expect(k8sClient.List(context.TODO(), snapList, client.InNamespace(ns1.Name))).To(Succeed())
				count := len(snapList.Items)
				Expect(k8sClient.List(context.TODO(), snapList, client.InNamespace(ns2.Name))).To(Succeed())
				count += len(snapList.Items)
				return count
			}, timeout, interval).Should(Equal(len(data)))
		})
	})
	When("a schedule has an expiration time", func() {
		It("should remove expired snapshots", func() {
			s := &snapschedulerv1.SnapshotSchedule{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "schedule",
					Namespace: ns1.Name,
				},
			}
			s.Spec.Retention.Expires = "24h"

			Expect(expireByTime(context.TODO(), s, time.Now(), logger, k8sClient)).To(Succeed())
			Eventually(func() int {
				snapList := &snapv1.VolumeSnapshotList{}
				Expect(k8sClient.List(context.TODO(), snapList, client.InNamespace(ns1.Name))).To(Succeed())
				count := len(snapList.Items)
				Expect(k8sClient.List(context.TODO(), snapList, client.InNamespace(ns2.Name))).To(Succeed())
				count += len(snapList.Items)
				return count
			}, timeout, interval).Should(Equal(len(data)))

			Expect(expireByTime(context.TODO(), s, time.Now().Add(48*time.Hour), logger, k8sClient)).To(Succeed())
			Eventually(func() int {
				snapList := &snapv1.VolumeSnapshotList{}
				Expect(k8sClient.List(context.TODO(), snapList, client.InNamespace(ns1.Name))).To(Succeed())
				count := len(snapList.Items)
				Expect(k8sClient.List(context.TODO(), snapList, client.InNamespace(ns2.Name))).To(Succeed())
				count += len(snapList.Items)
				return count
			}, timeout, interval).Should(Equal(len(data) - 1))
		})
	})
})

var _ = Describe("Grouping snapshots by PVC", func() {
	It("can group snapshots based on the PVC they were created from", func() {
		data := []struct {
			snapName string
			pvcName  string
		}{
			// testdata: s/^pvc/snap/ to get start of snap name
			{"snap1-1", "pvc1"},
			{"snap2-1", "pvc2"},
			{"snap1-2", "pvc1"},
			{"snap2-2", "pvc2"},
			{"snap3-blah", "pvc3"},
		}
		snapList := []snapv1.VolumeSnapshot{}
		for _, d := range data {
			pvcName := d.pvcName
			snapList = append(snapList, snapv1.VolumeSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name: d.snapName,
				},
				Spec: snapv1.VolumeSnapshotSpec{
					Source: snapv1.VolumeSnapshotSource{
						PersistentVolumeClaimName: &pvcName,
					},
				},
			})
		}
		// add one w/ nil Source
		snapList = append(snapList, snapv1.VolumeSnapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name: "i-have-nil-source",
			},
		})

		groupedSnaps := groupSnapsByPVC(snapList)
		wantSnaps := len(data)
		foundSnaps := 0
		for pvcName, list := range groupedSnaps {
			wantPrefix := strings.Replace(pvcName, "pvc", "snap", -1)
			for _, snap := range list {
				foundSnaps++
				Expect(snap.Name).To(HavePrefix(wantPrefix))
			}
		}
		Expect(wantSnaps).To(Equal(foundSnaps))
	})
})

var _ = Describe("Sorting snapshots", func() {
	It("can sort snapshots by time", func() {
		now := time.Now()
		inSnapList := []snapv1.VolumeSnapshot{
			{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Time{Time: now.Add(1 * time.Hour)},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Time{Time: now.Add(-1 * time.Hour)},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Time{Time: now},
				},
			},
		}
		outSnapList := sortSnapsByTime(inSnapList)
		Expect(len(outSnapList)).To(Equal(len(inSnapList)))
		Expect(outSnapList[0].CreationTimestamp.Before(&outSnapList[1].CreationTimestamp)).To(BeTrue())
		Expect(outSnapList[1].CreationTimestamp.Before(&outSnapList[2].CreationTimestamp)).To(BeTrue())

		Expect(sortSnapsByTime(nil)).To(BeNil())
	})
})

var _ = Describe("Deleting snapshots", func() {
	var ns1 *v1.Namespace
	var ns2 *v1.Namespace
	BeforeEach(func() {
		ns1 = &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
			},
		}
		Expect(k8sClient.Create(context.TODO(), ns1)).To(Succeed())
		Expect(ns1.Name).NotTo(BeEmpty())
		ns2 = &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
			},
		}
		Expect(k8sClient.Create(context.TODO(), ns2)).To(Succeed())
		Expect(ns2.Name).NotTo(BeEmpty())
	})
	AfterEach(func() {
		Expect(k8sClient.Delete(context.TODO(), ns1)).To(Succeed())
		Expect(k8sClient.Delete(context.TODO(), ns2)).To(Succeed())
	})
	It("deletes snapshots in the provided list", func() {
		snaps := []snapv1.VolumeSnapshot{
			{ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: ns1.Name,
			}},
			{ObjectMeta: metav1.ObjectMeta{
				Name:      "bar",
				Namespace: ns1.Name,
			}},
			{ObjectMeta: metav1.ObjectMeta{
				Name:      "baz",
				Namespace: ns2.Name,
			}},
			{ObjectMeta: metav1.ObjectMeta{
				Name:      "splat",
				Namespace: ns2.Name,
			}},
		}
		snapList := []snapv1.VolumeSnapshot{}
		snapList = append(snapList, snaps[1])
		snapList = append(snapList, snaps[2])

		for _, o := range snaps {
			Expect(k8sClient.Create(context.TODO(), &o)).To(Succeed())
		}

		Expect(deleteSnapshots(context.TODO(), snapList, logger, k8sClient)).To(Succeed())

		snap := &snapv1.VolumeSnapshot{}
		Eventually(func() bool {
			err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: "bar", Namespace: ns1.Name}, snap)
			return err != nil && kerrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())

		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKey{Name: "splat", Namespace: ns2.Name}, snap)
		}, timeout, interval).Should(Succeed())

		Expect(deleteSnapshots(context.TODO(), nil, logger, k8sClient)).To(Succeed())
	})
})

var _ = Describe("Expiring snapshots by count", func() {
	var ns1 *v1.Namespace
	var ns2 *v1.Namespace
	var data []struct {
		namespace string
		schedule  string
		pvcName   string
	}
	BeforeEach(func() {
		ns1 = &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
			},
		}
		Expect(k8sClient.Create(context.TODO(), ns1)).To(Succeed())
		Expect(ns1.Name).NotTo(BeEmpty())
		ns2 = &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
			},
		}
		Expect(k8sClient.Create(context.TODO(), ns2)).To(Succeed())
		Expect(ns2.Name).NotTo(BeEmpty())

		data = []struct {
			namespace string
			schedule  string
			pvcName   string
		}{
			{ns1.Name, "schedule", "pvc1"}, // this one will be deleted
			{ns1.Name, "schedule", "pvc1"},
			{ns1.Name, "schedule", "pvc1"},
			{ns1.Name, "schedule", "pvc1"},
			{ns2.Name, "schedule", "pvc1"},      // diff namespace, no match
			{ns1.Name, "schedule", "different"}, // diff pvc, only 1 of these
			{ns1.Name, "different", "pvc1"},     // diff schedule, no match
		}
		for _, d := range data {
			pvcName := d.pvcName
			snap := snapv1.VolumeSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      d.namespace + "-" + d.schedule + "-" + time.Now().Format("20060102150405"),
					Namespace: d.namespace,
					Labels: map[string]string{
						"foo":       "bar",
						ScheduleKey: d.schedule,
					},
				},
				Spec: snapv1.VolumeSnapshotSpec{
					Source: snapv1.VolumeSnapshotSource{
						PersistentVolumeClaimName: &pvcName,
					},
				},
			}
			Expect(k8sClient.Create(context.TODO(), &snap)).To(Succeed())
			time.Sleep(time.Second)
			Eventually(func() error {
				s := snapv1.VolumeSnapshot{}
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(&snap), &s)
			}, timeout, interval).Should(Succeed())
		}
	})
	AfterEach(func() {
		Expect(k8sClient.Delete(context.TODO(), ns1)).To(Succeed())
		Expect(k8sClient.Delete(context.TODO(), ns2)).To(Succeed())
	})

	It("doesn't delete any when there's no max", func() {
		noexpire := &snapschedulerv1.SnapshotSchedule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "schedule",
				Namespace: ns1.Name,
			},
		}
		// no maxCount, none should be pruned
		Expect(expireByCount(context.TODO(), noexpire, logger, k8sClient)).To(Succeed())
		Eventually(func() int {
			snapList := &snapv1.VolumeSnapshotList{}
			Expect(k8sClient.List(context.TODO(), snapList, client.InNamespace(ns1.Name))).To(Succeed())
			count := len(snapList.Items)
			Expect(k8sClient.List(context.TODO(), snapList, client.InNamespace(ns2.Name))).To(Succeed())
			count += len(snapList.Items)
			return count
		}, timeout, interval).Should(Equal(len(data)))
	})
	It("removes the oldest when there are too many", func() {
		s := &snapschedulerv1.SnapshotSchedule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "schedule",
				Namespace: ns1.Name,
			},
		}
		maxCount := int32(3)
		s.Spec.Retention.MaxCount = &maxCount

		Expect(expireByCount(context.TODO(), s, logger, k8sClient)).To(Succeed())
		Eventually(func() int {
			snapList := &snapv1.VolumeSnapshotList{}
			Expect(k8sClient.List(context.TODO(), snapList, client.InNamespace(ns1.Name))).To(Succeed())
			count := len(snapList.Items)
			Expect(k8sClient.List(context.TODO(), snapList, client.InNamespace(ns2.Name))).To(Succeed())
			count += len(snapList.Items)
			return count
		}, timeout, interval).Should(Equal(len(data) - 1))
	})
})
