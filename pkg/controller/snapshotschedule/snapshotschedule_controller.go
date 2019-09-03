package snapshotschedule

import (
	"context"
	"fmt"
	"time"

	snapschedulerv1alpha1 "github.com/backube/snap-scheduler/pkg/apis/snapscheduler/v1alpha1"
	snapv1alpha1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	"github.com/robfig/cron"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	// "k8s.io/apimachinery/pkg/types"
	// "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	maxRequeueTime = 60 * time.Second
	// SchedulerKey is a label applied to every snapshot created by
	// SnapScheduler
	SchedulerKey = "snapscheduler.backube/schedule"
	// Format for all time strings
	timeFormat = time.RFC3339
)

var log = logf.Log.WithName("controller_snapshotschedule")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new SnapshotSchedule Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileSnapshotSchedule{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("snapshotschedule-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource SnapshotSchedule
	err = c.Watch(&source.Kind{Type: &snapschedulerv1alpha1.SnapshotSchedule{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner SnapshotSchedule
	// err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
	// 	IsController: true,
	// 	OwnerType:    &snapschedulerv1alpha1.SnapshotSchedule{},
	// })
	// if err != nil {
	// 	return err
	// }

	return nil
}

// blank assignment to verify that ReconcileSnapshotSchedule implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileSnapshotSchedule{}

// ReconcileSnapshotSchedule reconciles a SnapshotSchedule object
type ReconcileSnapshotSchedule struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a SnapshotSchedule object and makes changes based on the state read
// and what is in the SnapshotSchedule.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileSnapshotSchedule) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling SnapshotSchedule")

	// Fetch the SnapshotSchedule instance
	instance := &snapschedulerv1alpha1.SnapshotSchedule{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// If necessary, initialize time of next snap based on schedule
	if instance.Status.NextSnapshotTime == "" {
		// Update nextSnapshot time based on current time and cronspec
		if err = updateNextSnapTime(instance, time.Now()); err != nil {
			reqLogger.Error(err, "couldn't update next snap time",
				"cronspec", instance.Spec.Schedule)
			return reconcile.Result{}, err
		}
	}

	timeNext, err := time.Parse(timeFormat, instance.Status.NextSnapshotTime)
	if err != nil {
		reqLogger.Error(err, "unable to parse next snap time",
			"nextSnapshotTime", instance.Status.NextSnapshotTime)
		return reconcile.Result{}, err
	}

	timeNow := time.Now()
	if timeNow.After(timeNext) {
		reqLogger.Info("take snaps", "now", timeNow, "scheduled", timeNext)

		// Create snaps
		// Update lastSnapshotTime

		// Taking snapshots requires some amount of time. Ensure timeNow
		// gets updated based on the end of the snapshot pass, not the
		// start of it.
		timeNow = time.Now()
	}

	// Update nextSnapshot time based on current time and cronspec
	if err = updateNextSnapTime(instance, timeNow); err != nil {
		reqLogger.Error(err, "couldn't update next snap time",
			"cronspec", instance.Spec.Schedule)
		return reconcile.Result{}, err
	}

	// Update instance.Status
	err = r.client.Status().Update(context.TODO(), instance)
	return reconcile.Result{RequeueAfter: maxRequeueTime}, err
}

func updateNextSnapTime(snapshotSchedule *snapschedulerv1alpha1.SnapshotSchedule, referenceTime time.Time) error {
	if snapshotSchedule == nil {
		return fmt.Errorf("nil snapshotschedule instance")
	}
	next, err := getNextSnapTime(snapshotSchedule.Spec.Schedule, referenceTime)
	if err != nil {
		// Couldn't parse cronspec; clear the next snap time
		snapshotSchedule.Status.NextSnapshotTime = ""
	} else {
		snapshotSchedule.Status.NextSnapshotTime = next.Format(timeFormat)
	}
	return err
}

// newSnapForClaim returns a VolumeSnapshot object based on a PVC
func newSnapForClaim(namespace string, pvcName string, snapName string, scheduleName string, labels map[string]string, snapClass *string) *snapv1alpha1.VolumeSnapshot {
	numLabels := 1
	if labels != nil {
		numLabels += len(labels)
	}
	snapLabels := make(map[string]string, numLabels)
	for k, v := range labels {
		snapLabels[k] = v
	}
	snapLabels[SchedulerKey] = scheduleName
	// https://github.com/kubernetes-csi/external-snapshotter/blob/master/pkg/apis/volumesnapshot/v1alpha1/types.go
	return &snapv1alpha1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      snapName,
			Namespace: namespace,
			Labels:    snapLabels,
		},
		Spec: snapv1alpha1.VolumeSnapshotSpec{
			Source: &corev1.TypedLocalObjectReference{
				APIGroup: nil,
				Kind:     "PersistentVolumeClaim",
				Name:     pvcName,
			},
			VolumeSnapshotClassName: snapClass,
		},
	}
}

/*
	// Define a new Pod object
	pod := newPodForCR(instance)

	// Set SnapshotSchedule instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Pod already exists
	found := &corev1.Pod{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
		err = r.client.Create(context.TODO(), pod)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Pod created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Pod already exists - don't requeue
	reqLogger.Info("Skip reconcile: Pod already exists", "Pod.Namespace", found.Namespace, "Pod.Name", found.Name)
	return reconcile.Result{}, nil
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newPodForCR(cr *snapschedulerv1alpha1.SnapshotSchedule) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}
}
*/

func getNextSnapTime(cronspec string, when time.Time) (time.Time, error) {
	schedule, err := cron.Parse(cronspec)
	if err != nil {
		return time.Time{}, err
	}

	next := schedule.Next(when)
	return next, nil
}
