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
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	snapschedulerv1 "github.com/backube/snapscheduler/api/v1"
)

const (
	timeFormat = time.RFC3339
)

var _ = DescribeTable("Determining the next snapshot time",
	func(cronspec string, current string, next string, expectErr bool) {
		ctime, _ := time.Parse(timeFormat, current)
		got, err := getNextSnapTime(cronspec, ctime)
		if expectErr {
			Expect(err).To(HaveOccurred())
		} else {
			want, _ := time.Parse(timeFormat, next)
			Expect(got).To(Equal(want))
		}
	},
	Entry("@hourly shortcut", "@hourly", "2013-02-01T11:04:05Z", "2013-02-01T12:00:00Z", false),
	Entry("01:02 on July 23 every year", "2 1 23 7 *", "2010-01-01T00:00:00Z", "2010-07-23T01:02:00Z", false),
	Entry("invalid spec", "invalid_spec", "2013-02-01T11:04:05Z", "unused", true),
)

var _ = Describe("newSnapForClaim", func() {
	It("creates a snapshot object based on a pvc, schedule, snapclass", func() {
		pvc := corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mypvc",
				Namespace: "mynamespace",
			},
		}
		snapname := "mysnap"
		snapClass := "snapclass"
		scheduleName := "mysched"
		schedTime, _ := time.Parse(timeFormat, "2010-07-23T01:02:00Z")
		snap := newSnapForClaim(snapname, pvc, scheduleName, schedTime, nil, &snapClass)

		Expect(snap.Name).To(Equal(snapname))
		Expect(snap.Namespace).To(Equal(pvc.Namespace))
		Expect(*snap.Spec.Source.PersistentVolumeClaimName).To(Equal(pvc.Name))
		Expect(snap.Spec.VolumeSnapshotClassName).NotTo(BeNil())
		Expect(*snap.Spec.VolumeSnapshotClassName).To(Equal(snapClass))
		Expect(snap.Labels).To(HaveKeyWithValue(ScheduleKey, scheduleName))
	})
	It("allows providing addl labels", func() {
		pvc := corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mypvc",
				Namespace: "mynamespace",
			},
		}
		snapname := "mysnap"
		scheduleName := "mysched"
		schedTime, _ := time.Parse(timeFormat, "2010-07-23T01:02:00Z")
		labels := make(map[string]string, 2)
		labels["one"] = "two"
		labels["three"] = "four"
		snap := newSnapForClaim(snapname, pvc, scheduleName, schedTime, labels, nil)
		// Some tests depend on knowing the internals :(
		Expect(snap.Spec.VolumeSnapshotClassName).To(BeNil())
		Expect(snap.Labels).NotTo(BeNil())
		Expect(snap.Labels).To(HaveKeyWithValue(ScheduleKey, scheduleName))
		Expect(snap.Labels).To(HaveKeyWithValue(WhenKey, schedTime.Format(timeYYYYMMDDHHMMSS)))
		Expect(snap.Labels).To(HaveKeyWithValue("three", "four"))
		Expect(len(snap.Labels)).To(Equal(4))
	})
})

var _ = Describe("UpdateNextSnapTime", func() {
	It("A nil schedule should generate an error", func() {
		Expect(updateNextSnapTime(nil, time.Now())).NotTo(Succeed())
	})
	It("An empty cronspec should generate an error", func() {
		s := &snapschedulerv1.SnapshotSchedule{}
		Expect(updateNextSnapTime(s, time.Now())).NotTo(Succeed())
	})
	It("should generate the correct next time", func() {
		s := &snapschedulerv1.SnapshotSchedule{}
		s.Spec.Schedule = "2 1 23 7 *"
		cTime, _ := time.Parse(timeFormat, "2010-01-01T00:00:00Z")
		Expect(updateNextSnapTime(s, cTime)).To(Succeed())
		expected, _ := time.Parse(timeFormat, "2010-07-23T01:02:00Z")
		Expect(s.Status.NextSnapshotTime.Time).To(Equal(expected))
	})
})

var _ = DescribeTable("Snapshot name generation",
	func(pvcName string, schedName string) {
		// https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/identifiers.md
		const maxAllowedNameLength = 253

		sName := snapshotName(pvcName, schedName, time.Now())
		Expect(len(sName)).To(BeNumerically("<=", maxAllowedNameLength))

		plen := len(pvcName)
		if plen > 10 {
			plen = 10
		}
		Expect(sName).To(ContainSubstring(pvcName[0:plen]))

		slen := len(schedName)
		if slen > 10 {
			slen = 10
		}
		Expect(sName).To(ContainSubstring(schedName[0:slen]))
	},
	Entry("both names are short", "foo", "bar"),
	Entry("PVC name is long", strings.Repeat("x", 250), "blah"),
	Entry("Schedule name is long", "blah", strings.Repeat("y", 250)),
	Entry("both names are long", strings.Repeat("x", 250), strings.Repeat("y", 250)),
)

func TestSnapshotName(t *testing.T) {
	data := []struct {
		pvcName   string
		schedName string
	}{
		// both are "short"
		{"foo", "bar"},
		// PVC name is long
		{strings.Repeat("x", 250), "blah"},
		// schedule name is long
		{"blah", strings.Repeat("y", 250)},
		// both are long
		{strings.Repeat("x", 250), strings.Repeat("y", 250)},
	}

	// https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/identifiers.md
	maxAllowedNameLength := 253

	for _, d := range data {
		sName := snapshotName(d.pvcName, d.schedName, time.Now())
		if len(sName) > maxAllowedNameLength {
			t.Errorf("snapshot name is too long. max: %v -- got: %v", maxAllowedNameLength, len(sName))
		}
		plen := len(d.pvcName)
		if plen > 10 {
			plen = 10
		}
		if !strings.Contains(sName, d.pvcName[0:plen]) {
			t.Errorf("Unable to find pvcName in snapshot name. snapshotName: %v -- pvcName: %v", sName, d.pvcName)
		}
		slen := len(d.schedName)
		if slen > 10 {
			slen = 10
		}
		if !strings.Contains(sName, d.schedName[0:slen]) {
			t.Errorf("Unable to find schedName in snapshot name. snapshotName: %v -- schedName: %v", sName, d.schedName)
		}
	}
}

var _ = Describe("Listing PVCs by selector", func() {
	var objects []corev1.PersistentVolumeClaim
	var ns *v1.Namespace
	BeforeEach(func() {
		ns = &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
			},
		}
		Expect(k8sClient.Create(context.TODO(), ns)).To(Succeed())
		Expect(ns.Name).NotTo(BeEmpty())

		objects = []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name-foo",
					Namespace: ns.Name,
					Labels: map[string]string{
						"mylabel": "foo",
					},
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							"storage": resource.MustParse("1Gi"),
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name-bar",
					Namespace: ns.Name,
					Labels: map[string]string{
						"mylabel": "bar",
					},
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							"storage": resource.MustParse("1Gi"),
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name-whatever",
					Namespace: ns.Name,
					Labels: map[string]string{
						"some": "label",
						"or":   "another",
					},
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							"storage": resource.MustParse("1Gi"),
						},
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
			Expect(k8sClient.Create(context.TODO(), &o)).To(Succeed())
			obj := corev1.PersistentVolumeClaim{}
			Eventually(func() error {
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(&o), &obj)
			}).Should(Succeed())
		}
	})
	It("can find PVCs by label selector", func() {
		mlFoo := &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"mylabel": "foo",
			},
		}
		pvcList, err := listPVCsMatchingSelector(context.TODO(), logger, k8sClient, ns.Name, mlFoo)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(pvcList.Items)).To(Equal(1))
		Expect(pvcList.Items[0].Name).To(Equal("name-foo"))
	})
	It("can find PVCs by match expression", func() {
		meBar := &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      "mylabel",
					Operator: metav1.LabelSelectorOpIn,
					Values: []string{
						"bar",
					},
				},
			},
		}
		pvcList, err := listPVCsMatchingSelector(context.TODO(), logger, k8sClient, ns.Name, meBar)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(pvcList.Items)).To(Equal(1))
		Expect(pvcList.Items[0].Name).To(Equal("name-bar"))
	})
	It("returns everything w/ an empty selector", func() {
		pvcList, err := listPVCsMatchingSelector(context.TODO(), logger, k8sClient, ns.Name, &metav1.LabelSelector{})
		Expect(err).NotTo(HaveOccurred())
		Expect(len(pvcList.Items)).To(Equal(len(objects)))
	})
})
