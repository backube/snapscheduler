package snapshotschedule

import (
	"testing"
	"time"
)

func TestGetNextSnapTime(t *testing.T) {
	var tests = []struct {
		inCronspec string
		inTime     string
		wantTime   string
		wantErr    bool
	}{
		{"@hourly", "2013-02-01T11:04:05Z", "2013-02-01T12:00:00Z", false},
		{"5 2 1 23 7 *", "2010-01-01T00:00:00Z", "2010-07-23T01:02:05Z", false},
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
	namespace := "mynamespace"
	pvc := "mypvc"
	snapname := "mysnap"
	snapClass := "snapclass"
	snap := newSnapForClaim(namespace, pvc, snapname, &snapClass)
	if snapname != snap.ObjectMeta.Name {
		t.Errorf("invalid snapshot name. expected: %v -- got: %v", snapname, snap.ObjectMeta.Name)
	}
	if namespace != snap.ObjectMeta.Namespace {
		t.Errorf("invalid snapshot namespace. expected: %v -- got: %v", namespace, snap.ObjectMeta.Namespace)
	}
	if pvc != snap.Spec.Source.Name {
		t.Errorf("invalid pvc name. expected: %v -- got: %v", pvc, snap.Spec.Source.Name)
	}
	if nil == snap.Spec.VolumeSnapshotClassName || snapClass != *snap.Spec.VolumeSnapshotClassName {
		t.Errorf("invalid snap class. expected: %v -- got: %v", snapClass, snap.Spec.VolumeSnapshotClassName)
	}

	snap = newSnapForClaim(namespace, pvc, snapname, nil)
	if nil != snap.Spec.VolumeSnapshotClassName {
		t.Errorf("expected nil snap class -- got: %v", snap.Spec.VolumeSnapshotClassName)
	}
}
