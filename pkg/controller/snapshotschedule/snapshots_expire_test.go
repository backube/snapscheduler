// nolint funlen  // Long test functions ok
package snapshotschedule

import (
	"context"
	"testing"
	"time"

	snapschedulerv1alpha1 "github.com/backube/snapscheduler/pkg/apis/snapscheduler/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	tlogr "github.com/go-logr/logr/testing"
	snapv1alpha1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func fakeClient(initialObjects []runtime.Object) client.Client {
	scheme := runtime.NewScheme()
	_ = snapschedulerv1alpha1.SchemeBuilder.AddToScheme(scheme)
	_ = snapv1alpha1.AddToScheme(scheme)
	return fake.NewFakeClientWithScheme(scheme, initialObjects...)
}

func TestGetExpirationTime(t *testing.T) {
	l := tlogr.NullLogger{}
	s := &snapschedulerv1alpha1.SnapshotSchedule{}

	// No retention time set
	expiration, err := getExpirationTime(s, time.Now(), l)
	if expiration != nil || err != nil {
		t.Errorf("empty spec.retention.expires. expected: nil,nil -- got: %v,%v", expiration, err)
	}

	// Unparsable retention time
	s.Spec.Retention.Expires = "garbage"
	_, err = getExpirationTime(s, time.Now(), l)
	if err == nil {
		t.Errorf("invalid spec.retention.expires. expected: error -- got: nil")
	}

	// Negative retention time
	s.Spec.Retention.Expires = "-10s"
	_, err = getExpirationTime(s, time.Now(), l)
	if err == nil {
		t.Errorf("negative spec.retention.expires. expected: error -- got: nil")
	}

	s.Spec.Retention.Expires = "1h"
	theTime, _ := time.Parse(timeFormat, "2013-02-01T11:04:05Z")
	expected := theTime.Add(-1 * time.Hour)
	expiration, err = getExpirationTime(s, theTime, l)
	if err != nil {
		t.Errorf("unexpected error return. expected: nil -- got: %v", err)
	}
	if expiration == nil || expected != *expiration {
		t.Errorf("incorrect expiration time. expected: %v -- got: %v", expected, expiration)
	}
}

func TestFilterExpiredSnaps(t *testing.T) {
	threshold, _ := time.Parse(timeFormat, "2000-01-01T00:00:00Z")
	times := []string{
		"1990-01-01T00:00:00Z", // expired
		"2010-02-10T10:30:05Z",
		"1999-12-31T23:59:00Z", // expired
		"2001-01-01T00:00:00Z",
		"2005-01-01T00:00:00Z",
	}
	expired := 2

	inList := &snapv1alpha1.VolumeSnapshotList{}
	for _, i := range times {
		theTime, _ := time.Parse(timeFormat, i)
		inList.Items = append(inList.Items, snapv1alpha1.VolumeSnapshot{
			ObjectMeta: metav1.ObjectMeta{
				CreationTimestamp: metav1.Time{
					Time: theTime,
				},
			},
		})
	}

	outList := filterExpiredSnaps(inList, threshold)
	if outList == nil {
		t.Error("unexpected nil output")
	}
	if len(outList.Items) != expired {
		t.Errorf("incorrect snapshots filtered. expected: %v -- got: %v", expired, len(outList.Items))
	}
}

func TestSnapshotsFromSchedule(t *testing.T) {
	l := tlogr.NullLogger{}
	objects := []runtime.Object{
		&snapv1alpha1.VolumeSnapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "default",
				Labels: map[string]string{
					"foo":       "bar",
					ScheduleKey: "s1",
				},
			},
		},
		&snapv1alpha1.VolumeSnapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "bar",
				Namespace: "default",
				Labels: map[string]string{
					"foo":       "bar",
					ScheduleKey: "s1",
				},
			},
		},
		&snapv1alpha1.VolumeSnapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "baz",
				Namespace: "default",
				Labels: map[string]string{
					"foo":       "bar",
					ScheduleKey: "s2",
				},
			},
		},
	}
	c := fakeClient(objects)
	s := &snapschedulerv1alpha1.SnapshotSchedule{}

	s.Name = "%%!! Invalid !!%%"
	_, err := snapshotsFromSchedule(s, l, c)
	if err == nil {
		t.Errorf("invalid schedule name should have produced an error")
	}

	s.Name = "s1"
	snapList, err := snapshotsFromSchedule(s, l, c)
	if err != nil {
		t.Errorf("unexpected error. got: %v", err)
	}
	if len(snapList.Items) != 2 {
		t.Errorf("matched wrong number of snapshots. expected: 2 -- got: %v", len(snapList.Items))
	}
	for _, snap := range snapList.Items {
		if snap.Name != "foo" && snap.Name != "bar" {
			t.Errorf("matched wrong snapshots. found: %v", snap.Name)
		}
	}
}

func TestExpireByTime(t *testing.T) {
	s := &snapschedulerv1alpha1.SnapshotSchedule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "schedule",
			Namespace: "same",
		},
	}
	s.Spec.Retention.Expires = "24h"

	noexpire := &snapschedulerv1alpha1.SnapshotSchedule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "schedule",
			Namespace: "same",
		},
	}

	now := time.Now()

	data := []struct {
		namespace   string
		created     time.Time
		schedule    string
		wantExpired bool
	}{
		{"same", now.Add(-1 * time.Hour), "schedule", false},
		{"different", now.Add(-1 * time.Hour), "schedule", false},
		{"same", now.Add(-48 * time.Hour), "schedule", true},
		{"different", now.Add(-48 * time.Hour), "schedule", false},
		{"same", now.Add(-1 * time.Hour), "different", false},
		{"different", now.Add(-1 * time.Hour), "different", false},
		{"same", now.Add(-48 * time.Hour), "different", false},
		{"different", now.Add(-48 * time.Hour), "different", false},
	}
	var objects []runtime.Object
	for _, d := range data {
		objects = append(objects, &snapv1alpha1.VolumeSnapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name:              d.namespace + "-" + d.schedule + "-" + d.created.Format("200601021504"),
				Namespace:         d.namespace,
				CreationTimestamp: metav1.Time{Time: d.created},
				Labels: map[string]string{
					"foo":       "bar",
					ScheduleKey: d.schedule,
				},
			},
		})
	}

	c := fakeClient(objects)
	l := tlogr.NullLogger{}

	err := expireByTime(noexpire, l, c)
	if err != nil {
		t.Errorf("unexpected error. got: %v", err)
	}
	snapList := &snapv1alpha1.VolumeSnapshotList{}
	_ = c.List(context.TODO(), &client.ListOptions{}, snapList)
	if len(snapList.Items) != len(data) {
		t.Errorf("wrong number of snapshots remain. expected: %v -- got: %v", len(data), len(snapList.Items))
	}

	err = expireByTime(s, l, c)
	if err != nil {
		t.Errorf("unexpected error. got: %v", err)
	}
	snapList = &snapv1alpha1.VolumeSnapshotList{}
	_ = c.List(context.TODO(), &client.ListOptions{}, snapList)
	if len(snapList.Items) != len(data)-1 {
		t.Errorf("wrong number of snapshots remain. expected: %v -- got: %v", len(data)-1, len(snapList.Items))
	}
}
