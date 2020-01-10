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
package snapshotschedule

import (
	"strings"
	"testing"
	"time"

	snapschedulerv1 "github.com/backube/snapscheduler/pkg/apis/snapscheduler/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	timeFormat = time.RFC3339
)

func TestGetNextSnapTime(t *testing.T) {
	var tests = []struct {
		inCronspec string
		inTime     string
		wantTime   string
		wantErr    bool
	}{
		{"@hourly", "2013-02-01T11:04:05Z", "2013-02-01T12:00:00Z", false},
		{"2 1 23 7 *", "2010-01-01T00:00:00Z", "2010-07-23T01:02:00Z", false},
		{"invalid_spec", "2013-02-01T11:04:05Z", "unused", true},
	}

	for _, test := range tests {
		inTime, _ := time.Parse(timeFormat, test.inTime)
		gotTime, err := getNextSnapTime(test.inCronspec, inTime)
		if err != nil {
			if !test.wantErr {
				t.Errorf("unexpected error: %v", err)
			}
			continue
		}
		if test.wantErr {
			t.Errorf("expected an error, but didn't get it")
			continue
		}
		wantTime, _ := time.Parse(timeFormat, test.wantTime)
		if gotTime != wantTime {
			t.Errorf("expected: %v -- got: %v", wantTime, gotTime)
		}
	}
}

func TestNewSnapForClaimV1beta1(t *testing.T) {
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
	VersionChecker.v1Beta1 = true
	VersionChecker.v1Alpha1 = false
	snap := newSnapForClaim(snapname, pvc, scheduleName, schedTime, nil, &snapClass)

	snapMeta := snap.ObjectMeta()
	// Some tests depend on knowing the internals :(
	betaSnap := snap.v1Beta1

	if snapname != snapMeta.Name {
		t.Errorf("invalid snapshot name. expected: %v -- got: %v", snapname, snapMeta.Name)
	}
	if pvc.Namespace != snapMeta.Namespace {
		t.Errorf("invalid snapshot namespace. expected: %v -- got: %v", pvc.Namespace, snapMeta.Namespace)
	}
	if pvc.Name != *betaSnap.Spec.Source.PersistentVolumeClaimName {
		t.Errorf("invalid pvc name. expected: %v -- got: %v", pvc.Name, betaSnap.Spec.Source.PersistentVolumeClaimName)
	}
	if nil == betaSnap.Spec.VolumeSnapshotClassName || snapClass != *betaSnap.Spec.VolumeSnapshotClassName {
		t.Errorf("invalid snap class. expected: %v -- got: %v", snapClass, betaSnap.Spec.VolumeSnapshotClassName)
	}
	if snapMeta.Labels == nil || scheduleName != snapMeta.Labels[ScheduleKey] {
		t.Errorf("SchedulerKey not found in snapshot labels")
	}

	labels := make(map[string]string, 2)
	labels["one"] = "two"
	labels["three"] = "four"
	snap = newSnapForClaim(snapname, pvc, scheduleName, schedTime, labels, nil)
	// Some tests depend on knowing the internals :(
	betaSnap = snap.v1Beta1
	if nil != betaSnap.Spec.VolumeSnapshotClassName {
		t.Errorf("expected nil snap class -- got: %v", betaSnap.Spec.VolumeSnapshotClassName)
	}
	snapMeta = snap.ObjectMeta()
	if snapMeta.Labels == nil {
		t.Errorf("unexpected nil set of labels")
	} else {
		if scheduleName != snapMeta.Labels[ScheduleKey] {
			t.Errorf("Wrong SchedulerKey in snapshot labels. expected: %v -- got: %v", scheduleName, snapMeta.Labels[ScheduleKey])
		}
		if schedTime.Format(timeYYYYMMDDHHMMSS) != snapMeta.Labels[WhenKey] {
			t.Errorf("Wrong WhenKey in snapshot labels. expected: %v -- got: %v",
				schedTime.Format(timeYYYYMMDDHHMMSS), snapMeta.Labels[WhenKey])
		}
		if "four" != snapMeta.Labels["three"] {
			t.Errorf("labels are not properly passed through")
		}
		numLabels := len(snapMeta.Labels)
		if numLabels != 4 {
			t.Errorf("unexpected number of labels. expected: 4 -- got: %v", numLabels)
		}
	}
}

func TestNewSnapForClaimV1alpha1(t *testing.T) {
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
	VersionChecker.v1Beta1 = false
	VersionChecker.v1Alpha1 = true
	snap := newSnapForClaim(snapname, pvc, scheduleName, schedTime, nil, &snapClass)

	snapMeta := snap.ObjectMeta()
	// Some tests depend on knowing the internals :(
	alphaSnap := snap.v1Alpha1

	if snapname != snapMeta.Name {
		t.Errorf("invalid snapshot name. expected: %v -- got: %v", snapname, snapMeta.Name)
	}
	if pvc.Namespace != snapMeta.Namespace {
		t.Errorf("invalid snapshot namespace. expected: %v -- got: %v", pvc.Namespace, snapMeta.Namespace)
	}
	if pvc.Name != alphaSnap.Spec.Source.Name {
		t.Errorf("invalid pvc name. expected: %v -- got: %v", pvc.Name, alphaSnap.Spec.Source.Name)
	}
	if nil == alphaSnap.Spec.VolumeSnapshotClassName || snapClass != *alphaSnap.Spec.VolumeSnapshotClassName {
		t.Errorf("invalid snap class. expected: %v -- got: %v", snapClass, alphaSnap.Spec.VolumeSnapshotClassName)
	}
	if snapMeta.Labels == nil || scheduleName != snapMeta.Labels[ScheduleKey] {
		t.Errorf("SchedulerKey not found in snapshot labels")
	}

	labels := make(map[string]string, 2)
	labels["one"] = "two"
	labels["three"] = "four"
	snap = newSnapForClaim(snapname, pvc, scheduleName, schedTime, labels, nil)
	// Some tests depend on knowing the internals :(
	alphaSnap = snap.v1Alpha1
	if nil != alphaSnap.Spec.VolumeSnapshotClassName {
		t.Errorf("expected nil snap class -- got: %v", alphaSnap.Spec.VolumeSnapshotClassName)
	}
	snapMeta = snap.ObjectMeta()
	if snapMeta.Labels == nil {
		t.Errorf("unexpected nil set of labels")
	} else {
		if scheduleName != snapMeta.Labels[ScheduleKey] {
			t.Errorf("Wrong SchedulerKey in snapshot labels. expected: %v -- got: %v", scheduleName, snapMeta.Labels[ScheduleKey])
		}
		if schedTime.Format(timeYYYYMMDDHHMMSS) != snapMeta.Labels[WhenKey] {
			t.Errorf("Wrong WhenKey in snapshot labels. expected: %v -- got: %v",
				schedTime.Format(timeYYYYMMDDHHMMSS), snapMeta.Labels[WhenKey])
		}
		if "four" != snapMeta.Labels["three"] {
			t.Errorf("labels are not properly passed through")
		}
		numLabels := len(snapMeta.Labels)
		if numLabels != 4 {
			t.Errorf("unexpected number of labels. expected: 4 -- got: %v", numLabels)
		}
	}
}

func TestUpdateNextSnapTime(t *testing.T) {
	err := updateNextSnapTime(nil, time.Now())
	if err == nil {
		t.Error("nil schedule should generate an error")
	}

	s := &snapschedulerv1.SnapshotSchedule{}
	err = updateNextSnapTime(s, time.Now())
	if err == nil {
		t.Error("empty cronspec should generate an error")
	}

	s.Spec.Schedule = "2 1 23 7 *"
	cTime, _ := time.Parse(timeFormat, "2010-01-01T00:00:00Z")
	err = updateNextSnapTime(s, cTime)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expected, _ := time.Parse(timeFormat, "2010-07-23T01:02:00Z")
	if s.Status.NextSnapshotTime.Time != expected {
		t.Errorf("incorrect next snap time. expected: %v -- got: %v", expected, s.Status.NextSnapshotTime)
	}
}

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

func TestListPVCSelector(t *testing.T) {
	objects := []runtime.Object{
		&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name-foo",
				Namespace: "mynamespace",
				Labels: map[string]string{
					"mylabel": "foo",
				},
			},
		},
		&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name-bar",
				Namespace: "mynamespace",
				Labels: map[string]string{
					"mylabel": "bar",
				},
			},
		},
		&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name-whatever",
				Namespace: "mynamespace",
				Labels: map[string]string{
					"some": "label",
					"or":   "another",
				},
			},
		},
	}
	c := fakeClient(objects)
	mlFoo := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"mylabel": "foo",
		},
	}
	pvcList, err := listPVCsMatchingSelector(nullLogger, c, "mynamespace", mlFoo)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(pvcList.Items) != 1 || pvcList.Items[0].Name != "name-foo" {
		t.Errorf("failed to find correct PVCs using matchlabels. expected: name-foo -- got: %v", pvcList.Items)
	}

	meBar := &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			metav1.LabelSelectorRequirement{
				Key:      "mylabel",
				Operator: metav1.LabelSelectorOpIn,
				Values: []string{
					"bar",
				},
			},
		},
	}
	pvcList, err = listPVCsMatchingSelector(nullLogger, c, "mynamespace", meBar)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(pvcList.Items) != 1 || pvcList.Items[0].Name != "name-bar" {
		t.Errorf("failed to find correct PVCs using matchexpressions. expected: name-bar -- got: %v", pvcList.Items)
	}

	pvcList, err = listPVCsMatchingSelector(nullLogger, c, "mynamespace", &metav1.LabelSelector{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(pvcList.Items) != len(objects) {
		t.Errorf("empty selector should have returned all items. expected:%v -- got: %v", len(objects), len(pvcList.Items))
	}
}
