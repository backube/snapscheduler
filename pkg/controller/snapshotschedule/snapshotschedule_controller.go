package snapshotschedule

import (
	"context"
	"fmt"
	"time"

	snapschedulerv1alpha1 "github.com/backube/snapscheduler/pkg/apis/snapscheduler/v1alpha1"
	"github.com/go-logr/logr"
	snapv1alpha1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	cron "github.com/robfig/cron/v3"
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
	// Max amount of time between reconciling a schedule object
	maxRequeueTime = 60 * time.Second
	// ScheduleKey is a label applied to every snapshot created by
	// snap-scheduler, denoting the schedule that created it
	ScheduleKey = "snap-scheduler.backube/schedule"
	// WhenKey is a label applied to every snapshot created by
	// snap-scheduler, denoting the scheduled (not actual) time of the snapshot
	WhenKey = "snap-scheduler.backube/when"
)

var log = logf.Log.WithName("controller_snapshotschedule")

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
func (r *ReconcileSnapshotSchedule) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling SnapshotSchedule")

	// Fetch the SnapshotSchedule instance
	instance := &snapschedulerv1alpha1.SnapshotSchedule{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// If necessary, initialize time of next snap based on schedule
	if instance.Status.NextSnapshotTime.IsZero() {
		// Update nextSnapshot time based on current time and cronspec
		if err = updateNextSnapTime(instance, time.Now()); err != nil {
			reqLogger.Error(err, "couldn't update next snap time",
				"cronspec", instance.Spec.Schedule)
			return reconcile.Result{}, err
		}
	}

	var result reconcile.Result
	if instance.Status.State == snapschedulerv1alpha1.StateUnknown {
		// Unknown state immediately transitions to Idle
		instance.Status.State = snapschedulerv1alpha1.StateIdle
		result = reconcile.Result{Requeue: true}
		err = nil
	} else if instance.Status.State == snapschedulerv1alpha1.StateIdle {
		result, err = r.handleIdle(instance, reqLogger)
		// result = reconcile.Result{RequeueAfter: maxRequeueTime}
		// err = nil
	} else if instance.Status.State == snapschedulerv1alpha1.StateSnapshotting {
		result, err = r.handleSnapshotting(instance, reqLogger)
	}

	// Update instance.Status
	err2 := r.client.Status().Update(context.TODO(), instance)
	if err == nil { // Don't mask previous error
		err = err2
	}
	return result, err
}

/*
	// Create snapshots
	if timeNow.After(timeNext) && !instance.Spec.Disabled {
		completed, err := createSnapshots(reqLogger, r.client, instance)
		if err != nil {
			return reconcile.Result{}, err
		}
		if completed {
			// Update lastSnapshotTime
			instance.Status.LastSnapshotTime.Time = time.Now()

			// Taking snapshots requires some amount of time. Ensure timeNow
			// gets updated based on the end of the snapshot pass, not the
			// start of it.
			timeNow = time.Now()
		}
	}

	// Update nextSnapshot time based on current time and cronspec
	if err = updateNextSnapTime(instance, timeNow); err != nil {
		reqLogger.Error(err, "couldn't update next snap time",
			"cronspec", instance.Spec.Schedule)
		return reconcile.Result{}, err
	}

	// Purge old snapshots
	if !instance.Spec.Disabled {
		// Delete any that are older than timeNow - Expires
		// Delete the oldest until total count <= MaxCount
	}

	// Update instance.Status
	err = r.client.Status().Update(context.TODO(), instance)
	return reconcile.Result{RequeueAfter: maxRequeueTime}, err
}
*/

func (r *ReconcileSnapshotSchedule) handleIdle(schedule *snapschedulerv1alpha1.SnapshotSchedule, logger logr.Logger) (reconcile.Result, error) {
	timeNow := time.Now()
	timeNext := schedule.Status.NextSnapshotTime.Time

	if !schedule.Spec.Disabled && timeNow.After(timeNext) {
		// It's time to take snaps... switch state
		schedule.Status.State = snapschedulerv1alpha1.StateSnapshotting
		return reconcile.Result{Requeue: true}, nil
	}

	// We always update nextSnapshot in case the schedule changed
	if err := updateNextSnapTime(schedule, timeNow); err != nil {
		logger.Error(err, "couldn't update next snap time",
			"cronspec", schedule.Spec.Schedule)
		return reconcile.Result{}, err
	}

	// Ensure we requeue in time for the next scheduled snapshot time
	durTillNext := timeNext.Sub(timeNow)
	requeueTime := maxRequeueTime
	if durTillNext < requeueTime {
		requeueTime = durTillNext
	}
	return reconcile.Result{RequeueAfter: requeueTime}, nil
}

func (r *ReconcileSnapshotSchedule) handleSnapshotting(schedule *snapschedulerv1alpha1.SnapshotSchedule, logger logr.Logger) (reconcile.Result, error) {
	// We're done taking snapshots. Transition back to idle
	schedule.Status.State = snapschedulerv1alpha1.StateIdle
	// Update lastSnapshot & nextSnapshot times
	timeNow := metav1.Now()
	schedule.Status.LastSnapshotTime = &timeNow
	if err := updateNextSnapTime(schedule, timeNow.Time); err != nil {
		logger.Error(err, "couldn't update next snap time",
			"cronspec", schedule.Spec.Schedule)
		return reconcile.Result{}, err
	}
	return reconcile.Result{Requeue: true}, nil
}

func createSnapshots(logger logr.Logger, c client.Client, schedule *snapschedulerv1alpha1.SnapshotSchedule) (bool, error) {
	logger.Info("taking scheduled snapshots", "scheduled", schedule.Status.NextSnapshotTime.Time)

	pvcl, err := listPVCsMatchingSelector(logger, c, schedule.Namespace, &schedule.Spec.ClaimSelector)
	if err != nil {
		logger.Error(err, "unable to get matching PVCs")
		return false, err
	}
	snapl, err := pvcListToSnapList(pvcl, nil)
	if err != nil {
		logger.Error(err, "unable to generate Snapshots from PVCs")
		// FIXME: This may result in infinite, rapid reconciling. We should back off.
		return false, err
	}
	logger.Info("creating Snapshot objects", "count", len(snapl.Items))
	for _, snapshot := range snapl.Items {
		err = c.Create(context.TODO(), &snapshot)
		if err != nil {
			logger.Error(err, "unable to create snapshot", "object", snapshot)
			return false, err
		}
	}
	return true, nil
}

func updateNextSnapTime(snapshotSchedule *snapschedulerv1alpha1.SnapshotSchedule, referenceTime time.Time) error {
	if snapshotSchedule == nil {
		return fmt.Errorf("nil snapshotschedule instance")
	}
	next, err := getNextSnapTime(snapshotSchedule.Spec.Schedule, referenceTime)
	if err != nil {
		// Couldn't parse cronspec; clear the next snap time
		snapshotSchedule.Status.NextSnapshotTime = nil
	} else {
		mv1time := metav1.NewTime(next)
		snapshotSchedule.Status.NextSnapshotTime = &mv1time
	}
	return err
}

// newSnapForClaim returns a VolumeSnapshot object based on a PVC
func newSnapForClaim(namespace string, pvcName string, snapName string, scheduleName string, labels map[string]string, snapClass *string) snapv1alpha1.VolumeSnapshot {
	numLabels := 2
	if labels != nil {
		numLabels += len(labels)
	}
	snapLabels := make(map[string]string, numLabels)
	for k, v := range labels {
		snapLabels[k] = v
	}
	snapLabels[ScheduleKey] = scheduleName
	snapLabels[WhenKey] = "FIXME"
	return snapv1alpha1.VolumeSnapshot{
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

func pvcListToSnapList(pvcList *corev1.PersistentVolumeClaimList, defaultSnapClass *string) (*snapv1alpha1.VolumeSnapshotList, error) {
	snapList := &snapv1alpha1.VolumeSnapshotList{}
	snapList.Items = make([]snapv1alpha1.VolumeSnapshot, len(pvcList.Items))
	for i, pvc := range pvcList.Items {
		snapList.Items[i] = newSnapForClaim(pvc.Namespace, pvc.Name, pvc.Name, "blah", pvc.Labels, defaultSnapClass)
	}
	return snapList, nil
}

// listPVCsMatchingSelector retrieves a list of PVCs that match the given selector
func listPVCsMatchingSelector(logger logr.Logger, c client.Client, namespace string, ls *metav1.LabelSelector) (*corev1.PersistentVolumeClaimList, error) {
	selector, err := metav1.LabelSelectorAsSelector(ls)
	if err != nil {
		return nil, err
	}
	pvcList := &corev1.PersistentVolumeClaimList{}
	err = c.List(context.TODO(), &client.ListOptions{LabelSelector: selector, Namespace: namespace}, pvcList)
	logger.Info("Created list of matching PVCs", "count", len(pvcList.Items))
	return pvcList, err
}

func parseCronspec(cronspec string) (cron.Schedule, error) {
	p := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	return p.Parse(cronspec)
}

func getNextSnapTime(cronspec string, when time.Time) (time.Time, error) {
	schedule, err := parseCronspec(cronspec)
	if err != nil {
		return time.Time{}, err
	}

	next := schedule.Next(when)
	return next, nil
}
