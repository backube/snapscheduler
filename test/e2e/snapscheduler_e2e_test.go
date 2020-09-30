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
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	rclient "sigs.k8s.io/controller-runtime/pkg/client"

	snapschedulerv1 "github.com/backube/snapscheduler/api/v1"
	sscontroller "github.com/backube/snapscheduler/controllers"
)

/*
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
*/
const (
	retryInterval = 5 * time.Second
	// Must be long enough for:
	// * snaps to be created via test schedule(s)
	// * snaps to become ready
	timeout         = 120 // seconds
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

func waitForPodReady(name string, namespace string) error {
	timeout := 5 * time.Minute
	err := wait.Poll(retryInterval, timeout, func() (bool, error) {
		pod := &corev1.Pod{}
		err := k8sClient.Get(goctx.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, pod)
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
func waitForSnapshot(client rclient.Reader, schedName string,
	namespace string, count int) ([]sscontroller.MultiversionSnapshot, error) {
	var snaps []sscontroller.MultiversionSnapshot
	timeout := 2 * time.Minute
	err := wait.Poll(retryInterval, timeout, func() (bool, error) {
		labelSelector := &metav1.LabelSelector{
			MatchLabels: map[string]string{
				sscontroller.ScheduleKey: schedName,
			},
		}
		selector, err := metav1.LabelSelectorAsSelector(labelSelector)
		Expect(err).NotTo(HaveOccurred())

		listOpts := []rclient.ListOption{
			rclient.InNamespace(namespace),
			rclient.MatchingLabelsSelector{
				Selector: selector,
			},
		}
		snaps, err = sscontroller.ListMVSnapshot(goctx.TODO(), client, listOpts...)
		Expect(err).NotTo(HaveOccurred())
		if len(snaps) < count {
			return false, nil
		}
		return true, nil
	})
	return snaps, err
}

func waitForSnapshotReady(client rclient.Reader, snapName string, namespace string) error {
	timeout := 2 * time.Minute
	err := wait.Poll(retryInterval, timeout, func() (bool, error) {
		snap, err := sscontroller.GetMVSnapshot(goctx.TODO(), client,
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

/*
func TestSnapscheduler(t *testing.T) {
	// Initialize MVSnapshot so we can work w/ both alpha and beta snaps
	if err := sscontroller.VersionChecker.SetConfig(cfg); err != nil {
		t.Fatalf("error setting version checker config: %v", err)
	}
	if err := sscontroller.VersionChecker.Refresh(tlogr.NullLogger{}); err != nil {
		t.Fatalf("initializing version checker: %v", err)
	}

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
*/
var _ = Describe("E2E tests", func() {
	var (
		namespace *corev1.Namespace
		pvc       corev1.PersistentVolumeClaim
	)

	BeforeEach(func() {
		namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "e2e-snapscheduler-",
			},
		}
		Expect(k8sClient.Create(goctx.TODO(), namespace)).To(Succeed())
		Expect(namespace.Name).ShouldNot(Equal(""))

		pvc = makePvc("pvc", namespace.Name, corev1.ReadWriteOnce, "1Gi", &storageClassName)
		Expect(k8sClient.Create(goctx.TODO(), &pvc)).To(Succeed())
	})

	AfterEach(func() {
		err := k8sClient.Delete(goctx.TODO(), namespace)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("When using a minimal schedule", func() {
		It("should generate a snapshot", func(done Done) {
			// Set up a PVC & pod to snapshot
			By("creating a pod")
			podName := "busybox"
			pod := makePod(podName, namespace.Name, pvc.GetName())
			Expect(k8sClient.Create(goctx.TODO(), &pod)).To(Succeed())

			By("waiting for pod to be running")
			Expect(waitForPodReady(podName, namespace.Name)).To(Succeed())

			By("creating a snapshot schedule")
			// Create a schedule
			schedName := "minimal"
			sched := snapschedulerv1.SnapshotSchedule{
				ObjectMeta: metav1.ObjectMeta{
					Name:      schedName,
					Namespace: namespace.Name,
				},
				Spec: snapschedulerv1.SnapshotScheduleSpec{
					Schedule: "* * * * *",
				},
			}
			Expect(k8sClient.Create(goctx.TODO(), &sched)).To(Succeed())

			By("waiting for a snapshot")
			snaps, err := waitForSnapshot(k8sClient, schedName, namespace.Name, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(snaps)).To(Equal(1))

			By("waiting for the snapshot to be ready")
			snapName := snaps[0].ObjectMeta().GetName()
			Expect(waitForSnapshotReady(k8sClient, snapName, namespace.Name)).To(Succeed())

			close(done)
		}, timeout)
	})

	Context("when setting labels in the template", func() {
		It("should set labels on the resulting snapshot", func(done Done) {
			By("creating a snapshot schedule w/ labels in the template")
			wantLabels := map[string]string{
				"mysnaplabel": "myval",
				"label2":      "v2",
			}
			schedName := "withlabels"
			sched := snapschedulerv1.SnapshotSchedule{
				ObjectMeta: metav1.ObjectMeta{
					Name:      schedName,
					Namespace: namespace.Name,
				},
				Spec: snapschedulerv1.SnapshotScheduleSpec{
					Schedule: "* * * * *",
					SnapshotTemplate: &snapschedulerv1.SnapshotTemplateSpec{
						Labels: wantLabels,
					},
				},
			}
			Expect(k8sClient.Create(goctx.TODO(), &sched)).To(Succeed())

			// Wait for a snapshot to be created
			By("waiting for snapshot to be created")
			snaps, err := waitForSnapshot(k8sClient, schedName, namespace.Name, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(snaps).To(HaveLen(1))

			gotLabels := snaps[0].ObjectMeta().GetLabels()
			for k, v := range wantLabels {
				Expect(gotLabels).To(HaveKeyWithValue(k, v))
			}

			close(done)
		}, timeout)
	})

	Context("when specifying a custom snapshotclass in the template", func() {
		It("should use that class in the resulting snapshot", func(done Done) {
			wantCustomClass := "my-custom-class"
			schedName := "class"
			sched := snapschedulerv1.SnapshotSchedule{
				ObjectMeta: metav1.ObjectMeta{
					Name:      schedName,
					Namespace: namespace.Name,
				},
				Spec: snapschedulerv1.SnapshotScheduleSpec{
					Schedule: "* * * * *",
					SnapshotTemplate: &snapschedulerv1.SnapshotTemplateSpec{
						SnapshotClassName: &wantCustomClass,
					},
				},
			}
			Expect(k8sClient.Create(goctx.TODO(), &sched)).To(Succeed())

			By("waiting for snapshot to be created")
			snaps, err := waitForSnapshot(k8sClient, schedName, namespace.Name, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(snaps).To(HaveLen(1))

			Expect(snaps[0].SnapshotClassName()).ToNot(BeNil())
			Expect(*snaps[0].SnapshotClassName()).To(Equal(wantCustomClass))

			close(done)
		}, timeout)
	})

	Context("when there are multiple PVCs", func() {
		It("should create multiple snapshots", func(done Done) {
			pvc2 := makePvc("second", namespace.Name, corev1.ReadWriteOnce, "1Gi", &storageClassName)
			Expect(k8sClient.Create(goctx.TODO(), &pvc2)).To(Succeed())

			schedName := "multi"
			sched := snapschedulerv1.SnapshotSchedule{
				ObjectMeta: metav1.ObjectMeta{
					Name:      schedName,
					Namespace: namespace.Name,
				},
				Spec: snapschedulerv1.SnapshotScheduleSpec{
					Schedule: "* * * * *",
				},
			}
			Expect(k8sClient.Create(goctx.TODO(), &sched)).To(Succeed())

			expectedSnaps := 2

			By("waiting for snapshots to be created")
			snaps, err := waitForSnapshot(k8sClient, schedName, namespace.Name, expectedSnaps)
			Expect(err).NotTo(HaveOccurred())
			Expect(snaps).To(HaveLen(expectedSnaps))

			close(done)
		}, timeout)
	})

	Context("when there are label selectors", func() {
		It("should only snapshot matching PVCs", func(done Done) {
			pvc := makePvc("pvc-yes", namespace.Name, corev1.ReadWriteOnce, "1Gi", &storageClassName)
			pvc.SetLabels(map[string]string{
				"snap":     "yes",
				"whatever": "zzz",
			})
			Expect(k8sClient.Create(goctx.TODO(), &pvc)).To(Succeed())

			pvc2 := makePvc("pvc-no", namespace.Name, corev1.ReadWriteOnce, "1Gi", &storageClassName)
			pvc2.SetLabels(map[string]string{
				"snap":     "no",
				"whatever": "zzz",
			})
			Expect(k8sClient.Create(goctx.TODO(), &pvc2)).To(Succeed())

			// Create a schedule
			schedName := "select"
			sched := snapschedulerv1.SnapshotSchedule{
				ObjectMeta: metav1.ObjectMeta{
					Name:      schedName,
					Namespace: namespace.Name,
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
			Expect(k8sClient.Create(goctx.TODO(), &sched)).To(Succeed())

			// Wait for first snapshot to be created
			By("waiting for snapshot to be created")
			snaps, err := waitForSnapshot(k8sClient, schedName, namespace.Name, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(snaps).To(HaveLen(1))
			Expect(snaps[0].ObjectMeta().GetName()).To(HavePrefix("pvc-yes"))

			close(done)
		}, timeout)
	})
})
