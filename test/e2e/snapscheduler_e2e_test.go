/*
Copyright (C) 2020  The snapscheduler authors

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

package e2e

import (
	goctx "context"
	"os"
	"strings"
	"testing"
	"time"

	tlogr "github.com/go-logr/logr/testing"
	snapv1alpha1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	snapv1beta1 "github.com/kubernetes-csi/external-snapshotter/v2/pkg/apis/volumesnapshot/v1beta1"
	sdktest "github.com/operator-framework/operator-sdk/pkg/test"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	rclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/backube/snapscheduler/pkg/apis"
	snapschedulerv1 "github.com/backube/snapscheduler/pkg/apis/snapscheduler/v1"
	"github.com/backube/snapscheduler/pkg/controller/snapshotschedule"
)

// The list of tests to run. This could probably be automated via some sort of
// reflect magic.
var testList = []struct {
	Name string
	Test func(t *testing.T)
}{
	{"Minimal schedule", minimalTest},
	{"Snapshot labeling", labelTest},
	{"Custom snapclass", customClassTest},
	{"Multiple PVCs", multiTest},
	{"PVC selector", selectorTest},
}

const (
	retryInterval = 5 * time.Second
	// Must be long enough for:
	// * snaps to be created via test schedule(s)
	// * snaps to become ready
	timeout         = 2 * time.Minute
	EnvStorageClass = "STORAGE_CLASS_NAME"
	EnvSnapClass    = "SNAPSHOT_CLASS_NAME"
)

var (
	storageClassName  = "csi-hostpath-sc"
	snapshotClassName = "csi-hostpath-snapclass"
)

func makePod(name string, namespace string, pvcName string) corev1.Pod {
	var gracePeriod int64 = 2
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep", "999999"},
					VolumeMounts: []corev1.VolumeMount{
						corev1.VolumeMount{
							Name:      "data",
							MountPath: "/mnt",
						},
					},
				},
			},
			TerminationGracePeriodSeconds: &gracePeriod,
			Volumes: []corev1.Volume{
				corev1.Volume{
					Name: "data",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
				},
			},
		},
	}
}

//nolint:unparam
func makePvc(name string, namespace string, mode corev1.PersistentVolumeAccessMode,
	size string, storageClassName *string) corev1.PersistentVolumeClaim {
	return corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				mode,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(size),
				},
			},
			StorageClassName: storageClassName,
		},
	}
}

func waitForPodReady(name string, namespace string, retryInterval time.Duration,
	timeout time.Duration) error {
	client := sdktest.Global.Client
	err := wait.Poll(retryInterval, timeout, func() (bool, error) {
		pod := &corev1.Pod{}
		err := client.Get(goctx.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, pod)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}

		if pod.Status.Phase == corev1.PodRunning {
			return true, nil
		}
		return false, nil
	})
	return err
}

//nolint:unparam
func waitForSnapshot(t *testing.T, client rclient.Reader, schedName string,
	namespace string, retryInterval time.Duration, timeout time.Duration,
	count int) ([]snapshotschedule.MultiversionSnapshot, error) {
	var snaps []snapshotschedule.MultiversionSnapshot
	err := wait.Poll(retryInterval, timeout, func() (bool, error) {
		labelSelector := &metav1.LabelSelector{
			MatchLabels: map[string]string{
				snapshotschedule.ScheduleKey: schedName,
			},
		}
		selector, err := metav1.LabelSelectorAsSelector(labelSelector)
		if err != nil {
			t.Errorf("unable to create label selector for snapshot: %v", err)
			return false, err
		}

		listOpts := []rclient.ListOption{
			rclient.InNamespace(namespace),
			rclient.MatchingLabelsSelector{
				Selector: selector,
			},
		}
		snaps, err = snapshotschedule.ListMVSnapshot(goctx.TODO(), client, listOpts...)
		if err != nil {
			t.Errorf("unable to list snapshots: %v", err)
			return false, err
		}
		if len(snaps) < count {
			return false, nil
		}
		return true, nil
	})
	return snaps, err
}

func waitForSnapshotReady(client rclient.Reader, snapName string, namespace string,
	retryInterval time.Duration, timeout time.Duration) error {
	err := wait.Poll(retryInterval, timeout, func() (bool, error) {
		snap, err := snapshotschedule.GetMVSnapshot(goctx.TODO(), client,
			types.NamespacedName{Name: snapName, Namespace: namespace})
		if err != nil {
			return false, err
		}

		ready := snap.ReadyToUse()
		if ready != nil && *ready {
			return true, nil
		}
		return false, nil
	})
	return err
}

func TestSnapscheduler(t *testing.T) {
	scheduleList := &snapschedulerv1.SnapshotScheduleList{}
	if err := sdktest.AddToFrameworkScheme(apis.AddToScheme, scheduleList); err != nil {
		t.Fatalf("unable to add scheme: %v", err)
	}
	_ = sdktest.AddToFrameworkScheme(snapv1alpha1.AddToScheme, &snapv1alpha1.VolumeSnapshotList{})
	_ = sdktest.AddToFrameworkScheme(snapv1beta1.AddToScheme, &snapv1beta1.VolumeSnapshotList{})

	// Initialize MVSnapshot so we can work w/ both alpha and beta snaps
	if err := snapshotschedule.VersionChecker.SetConfig(sdktest.Global.KubeConfig); err != nil {
		t.Fatalf("error setting version checker config: %v", err)
	}
	if err := snapshotschedule.VersionChecker.Refresh(tlogr.NullLogger{}); err != nil {
		t.Fatalf("initializing version checker: %v", err)
	}

	// Note, we don't set up the operator or wait for it to be ready.

	// Allow override of StorageClass and SnapshotClass names via environment
	// variables
	if v := os.Getenv(EnvStorageClass); v != "" {
		storageClassName = v
	}
	t.Logf("using StorageClassName: %v", storageClassName)
	if v := os.Getenv(EnvSnapClass); v != "" {
		snapshotClassName = v
	}
	t.Logf("using SnapshotClassName: %v", snapshotClassName)

	// run subtests
	for _, item := range testList {
		t.Run(item.Name, item.Test)
	}
}

//nolint:funlen
func minimalTest(t *testing.T) {
	t.Parallel()
	ctx := sdktest.NewTestCtx(t)
	defer ctx.Cleanup()

	cleanupOptions := sdktest.CleanupOptions{
		TestContext:   ctx,
		Timeout:       timeout,
		RetryInterval: retryInterval,
	}
	client := sdktest.Global.Client
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}

	// Set up a PVC & pod to snapshot
	pvc := makePvc("pvc", namespace, corev1.ReadWriteOnce, "1Gi", &storageClassName)
	if err = client.Create(goctx.TODO(), &pvc, &cleanupOptions); err != nil {
		t.Fatalf("creating pvc: %v", err)
	}
	podName := "busybox"
	pod := makePod(podName, namespace, pvc.GetName())
	if err = client.Create(goctx.TODO(), &pod, &cleanupOptions); err != nil {
		t.Fatalf("creating pod: %v", err)
	}
	t.Logf("waiting for pod %v/%v to be ready", namespace, podName)
	err = waitForPodReady(podName, namespace, retryInterval, timeout)
	if err != nil {
		t.Fatalf("pod failed to become ready: %v", err)
	}
	t.Logf("pod %v/%v is running", namespace, podName)

	// Create a schedule
	schedName := "minimal"
	sched := snapschedulerv1.SnapshotSchedule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      schedName,
			Namespace: namespace,
		},
		Spec: snapschedulerv1.SnapshotScheduleSpec{
			Schedule: "* * * * *",
		},
	}
	if err = client.Create(goctx.TODO(), &sched, &cleanupOptions); err != nil {
		t.Fatalf("creating snapshot schedule: %v", err)
	}

	// Wait for a snapshot to be created
	t.Log("waiting for snapshot to be created")
	snaps, err := waitForSnapshot(t, client, schedName, namespace, retryInterval, timeout, 1)
	if err != nil {
		t.Fatalf("waiting for snapshot: %v", err)
	}
	if len(snaps) != 1 {
		t.Fatalf("wrong number of snapshots. expected:1, got:%v", len(snaps))
	}

	// Wait for it to be ready
	snapName := snaps[0].ObjectMeta().GetName()
	t.Logf("found snapshot: %v/%v", namespace, snapName)
	err = waitForSnapshotReady(client, snapName, namespace, retryInterval, timeout)
	if err != nil {
		t.Fatalf("waiting for snapshot to be ready: %v", err)
	}
	t.Log("snapshot is ready")
}

//nolint:funlen
func labelTest(t *testing.T) {
	t.Parallel()
	ctx := sdktest.NewTestCtx(t)
	defer ctx.Cleanup()

	cleanupOptions := sdktest.CleanupOptions{
		TestContext:   ctx,
		Timeout:       timeout,
		RetryInterval: retryInterval,
	}
	client := sdktest.Global.Client
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}

	// Set up a PVC
	pvc := makePvc("pvc", namespace, corev1.ReadWriteOnce, "1Gi", &storageClassName)
	if err = client.Create(goctx.TODO(), &pvc, &cleanupOptions); err != nil {
		t.Fatalf("creating pvc: %v", err)
	}

	wantLabels := map[string]string{
		"mysnaplabel": "myval",
		"label2":      "v2",
	}

	// Create a schedule
	schedName := "withlabels"
	sched := snapschedulerv1.SnapshotSchedule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      schedName,
			Namespace: namespace,
		},
		Spec: snapschedulerv1.SnapshotScheduleSpec{
			Schedule: "* * * * *",
			SnapshotTemplate: &snapschedulerv1.SnapshotTemplateSpec{
				Labels: wantLabels,
			},
		},
	}
	if err = client.Create(goctx.TODO(), &sched, &cleanupOptions); err != nil {
		t.Fatalf("creating snapshot schedule: %v", err)
	}

	// Wait for a snapshot to be created
	t.Log("waiting for snapshot to be created")
	snaps, err := waitForSnapshot(t, client, schedName, namespace, retryInterval, timeout, 1)
	if err != nil {
		t.Fatalf("waiting for snapshot: %v", err)
	}
	if len(snaps) != 1 {
		t.Fatalf("wrong number of snapshots. expected:1, got:%v", len(snaps))
	}

	gotLabels := snaps[0].ObjectMeta().GetLabels()
	for k, v := range wantLabels {
		val, found := gotLabels[k]
		if !found || v != val {
			t.Errorf("unable to find label %v with value %v", k, v)
		}
	}
}

//nolint:funlen
func customClassTest(t *testing.T) {
	t.Parallel()
	ctx := sdktest.NewTestCtx(t)
	defer ctx.Cleanup()

	cleanupOptions := sdktest.CleanupOptions{
		TestContext:   ctx,
		Timeout:       timeout,
		RetryInterval: retryInterval,
	}
	client := sdktest.Global.Client
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}

	// Set up a PVC
	pvc := makePvc("pvc", namespace, corev1.ReadWriteOnce, "1Gi", &storageClassName)
	if err = client.Create(goctx.TODO(), &pvc, &cleanupOptions); err != nil {
		t.Fatalf("creating pvc: %v", err)
	}

	wantCustomClass := "my-custom-class"

	// Create a schedule
	schedName := "class"
	sched := snapschedulerv1.SnapshotSchedule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      schedName,
			Namespace: namespace,
		},
		Spec: snapschedulerv1.SnapshotScheduleSpec{
			Schedule: "* * * * *",
			SnapshotTemplate: &snapschedulerv1.SnapshotTemplateSpec{
				SnapshotClassName: &wantCustomClass,
			},
		},
	}
	if err = client.Create(goctx.TODO(), &sched, &cleanupOptions); err != nil {
		t.Fatalf("creating snapshot schedule: %v", err)
	}

	// Wait for a snapshot to be created
	t.Log("waiting for snapshot to be created")
	snaps, err := waitForSnapshot(t, client, schedName, namespace, retryInterval, timeout, 1)
	if err != nil {
		t.Fatalf("waiting for snapshot: %v", err)
	}
	if len(snaps) != 1 {
		t.Fatalf("wrong number of snapshots. expected:1, got:%v", len(snaps))
	}

	gotClass := snaps[0].SnapshotClassName()
	if gotClass == nil {
		t.Errorf("nil SnapshotClassName")
	} else if wantCustomClass != *gotClass {
		t.Errorf("wrong SnapshotClassName. want:%v got:%v", wantCustomClass, *gotClass)
	}
}

//nolint:funlen
func multiTest(t *testing.T) {
	t.Parallel()
	ctx := sdktest.NewTestCtx(t)
	defer ctx.Cleanup()

	cleanupOptions := sdktest.CleanupOptions{
		TestContext:   ctx,
		Timeout:       timeout,
		RetryInterval: retryInterval,
	}
	client := sdktest.Global.Client
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}

	// Set up a PVC
	pvc := makePvc("first", namespace, corev1.ReadWriteOnce, "1Gi", &storageClassName)
	if err = client.Create(goctx.TODO(), &pvc, &cleanupOptions); err != nil {
		t.Fatalf("creating pvc: %v", err)
	}

	pvc2 := makePvc("second", namespace, corev1.ReadWriteOnce, "1Gi", &storageClassName)
	if err = client.Create(goctx.TODO(), &pvc2, &cleanupOptions); err != nil {
		t.Fatalf("creating pvc: %v", err)
	}

	expectedSnaps := 2

	// Create a schedule
	schedName := "multi"
	sched := snapschedulerv1.SnapshotSchedule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      schedName,
			Namespace: namespace,
		},
		Spec: snapschedulerv1.SnapshotScheduleSpec{
			Schedule: "* * * * *",
		},
	}
	if err = client.Create(goctx.TODO(), &sched, &cleanupOptions); err != nil {
		t.Fatalf("creating snapshot schedule: %v", err)
	}

	// Wait for first snapshot to be created
	t.Log("waiting for snapshot to be created")
	snaps, err := waitForSnapshot(t, client, schedName, namespace, retryInterval, timeout, expectedSnaps)
	if err != nil {
		t.Fatalf("waiting for snapshot: %v", err)
	}
	if len(snaps) != expectedSnaps {
		t.Fatalf("wrong number of snapshots. expected:%v, got:%v", expectedSnaps, len(snaps))
	}
}

//nolint:funlen
func selectorTest(t *testing.T) {
	t.Parallel()
	ctx := sdktest.NewTestCtx(t)
	defer ctx.Cleanup()

	cleanupOptions := sdktest.CleanupOptions{
		TestContext:   ctx,
		Timeout:       timeout,
		RetryInterval: retryInterval,
	}
	client := sdktest.Global.Client
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}

	// Set up a PVC
	pvc := makePvc("pvc-yes", namespace, corev1.ReadWriteOnce, "1Gi", &storageClassName)
	pvc.SetLabels(map[string]string{
		"snap":     "yes",
		"whatever": "zzz",
	})
	if err = client.Create(goctx.TODO(), &pvc, &cleanupOptions); err != nil {
		t.Fatalf("creating pvc: %v", err)
	}

	pvc2 := makePvc("pvc-no", namespace, corev1.ReadWriteOnce, "1Gi", &storageClassName)
	pvc2.SetLabels(map[string]string{
		"snap":     "no",
		"whatever": "zzz",
	})
	if err = client.Create(goctx.TODO(), &pvc2, &cleanupOptions); err != nil {
		t.Fatalf("creating pvc: %v", err)
	}

	// Create a schedule
	schedName := "select"
	sched := snapschedulerv1.SnapshotSchedule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      schedName,
			Namespace: namespace,
		},
		Spec: snapschedulerv1.SnapshotScheduleSpec{
			Schedule: "* * * * *",
			ClaimSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"snap": "yes",
				},
			},
		},
	}
	if err = client.Create(goctx.TODO(), &sched, &cleanupOptions); err != nil {
		t.Fatalf("creating snapshot schedule: %v", err)
	}

	// Wait for first snapshot to be created
	t.Log("waiting for snapshot to be created")
	snaps, err := waitForSnapshot(t, client, schedName, namespace, retryInterval, timeout, 1)
	if err != nil {
		t.Fatalf("waiting for snapshot: %v", err)
	}
	if len(snaps) != 1 {
		t.Fatalf("wrong number of snapshots. expected:1, got:%v", len(snaps))
	}
	if !strings.Contains(snaps[0].ObjectMeta().GetName(), "pvc-yes") {
		t.Errorf("unexpected snapshot: %v", snaps[0].ObjectMeta().GetName())
	}
}
