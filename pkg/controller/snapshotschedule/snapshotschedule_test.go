package snapshotschedule

import (
	"testing"
	"time"

	snapschedulerv1alpha1 "github.com/backube/snapscheduler/pkg/apis/snapscheduler/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestNewSnapForClaim(t *testing.T) {
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
	if snapname != snap.Name {
		t.Errorf("invalid snapshot name. expected: %v -- got: %v", snapname, snap.Name)
	}
	if pvc.Namespace != snap.Namespace {
		t.Errorf("invalid snapshot namespace. expected: %v -- got: %v", pvc.Namespace, snap.Namespace)
	}
	if pvc.Name != snap.Spec.Source.Name {
		t.Errorf("invalid pvc name. expected: %v -- got: %v", pvc.Name, snap.Spec.Source.Name)
	}
	if nil == snap.Spec.VolumeSnapshotClassName || snapClass != *snap.Spec.VolumeSnapshotClassName {
		t.Errorf("invalid snap class. expected: %v -- got: %v", snapClass, snap.Spec.VolumeSnapshotClassName)
	}
	if snap.ObjectMeta.Labels == nil || scheduleName != snap.ObjectMeta.Labels[ScheduleKey] {
		t.Errorf("SchedulerKey not found in snapshot labels")
	}

	labels := make(map[string]string, 2)
	labels["one"] = "two"
	labels["three"] = "four"
	snap = newSnapForClaim(snapname, pvc, scheduleName, schedTime, labels, nil)
	if nil != snap.Spec.VolumeSnapshotClassName {
		t.Errorf("expected nil snap class -- got: %v", snap.Spec.VolumeSnapshotClassName)
	}
	if snap.Labels == nil {
		t.Errorf("unexpected nil set of labels")
	} else {
		if scheduleName != snap.Labels[ScheduleKey] {
			t.Errorf("Wrong SchedulerKey in snapshot labels. expected: %v -- got: %v", scheduleName, snap.Labels[ScheduleKey])
		}
		if schedTime.Format(timeYYYYMMDDHHMMSS) != snap.Labels[WhenKey] {
			t.Errorf("Wrong WhenKey in snapshot labels. expected: %v -- got: %v",
				schedTime.Format(timeYYYYMMDDHHMMSS), snap.Labels[WhenKey])
		}
		if "four" != snap.Labels["three"] {
			t.Errorf("labels are not properly passed through")
		}
		numLabels := len(snap.ObjectMeta.Labels)
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

	s := &snapschedulerv1alpha1.SnapshotSchedule{}
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
		t.Errorf("incorrect next snap time. expected %v -- got: %v", expected, s.Status.NextSnapshotTime)
	}
}
