package snapshotschedule

import (
	"context"
	"errors"
	"fmt"
	"time"

	snapschedulerv1alpha1 "github.com/backube/snapscheduler/pkg/apis/snapscheduler/v1alpha1"
	"github.com/go-logr/logr"
	snapv1alpha1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	cron "github.com/robfig/cron/v3"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
	maxRequeueTime = 5 * time.Minute
	// ScheduleKey is a label applied to every snapshot created by
	// snap-scheduler, denoting the schedule that created it
	ScheduleKey = "snapscheduler.backube/schedule"
	// Time format for snapshot names and labels
	timeYYYYMMDDHHMMSS = "200601021504"
	// WhenKey is a label applied to every snapshot created by
	// snap-scheduler, denoting the scheduled (not actual) time of the snapshot
	WhenKey = "snapscheduler.backube/when"
)

var log = logf.Log.WithName("controller_snapshotschedule")

// Add creates a new SnapshotSchedule Controller and adds it to the Manager.
// The Manager will set fields on the Controller and Start it when the Manager is Started.
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
		if kerrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	result, err := doReconcile(instance, reqLogger, r.client)

	// Update result in CR
	if err != nil {
		instance.Status.ReconcileResult = err.Error()
	} else {
		instance.Status.ReconcileResult = "ok"
	}

	// Update instance.Status
	err2 := r.client.Status().Update(context.TODO(), instance)
	if err == nil { // Don't mask previous error
		err = err2
	}
	return result, err
}

func doReconcile(schedule *snapschedulerv1alpha1.SnapshotSchedule,
	logger logr.Logger, c client.Client) (reconcile.Result, error) {
	var err error

	// If necessary, initialize time of next snap based on schedule
	if schedule.Status.NextSnapshotTime.IsZero() {
		// Update nextSnapshot time based on current time and cronspec
		if err = updateNextSnapTime(schedule, time.Now()); err != nil {
			logger.Error(err, "couldn't update next snap time",
				"cronspec", schedule.Spec.Schedule)
			return reconcile.Result{}, err
		}
	}

	// Dispatch based on State
	logger.Info("reconciling", "state", schedule.Status.State)
	var result reconcile.Result
	if schedule.Status.State == snapschedulerv1alpha1.StateUnknown {
		// Unknown state immediately transitions to Idle
		schedule.Status.State = snapschedulerv1alpha1.StateIdle
		result = reconcile.Result{Requeue: true}
		err = nil
	} else if schedule.Status.State == snapschedulerv1alpha1.StateIdle {
		result, err = handleIdle(schedule, logger, c)
	} else if schedule.Status.State == snapschedulerv1alpha1.StateSnapshotting {
		result, err = handleSnapshotting(schedule, logger, c)
	}

	return result, err
}

func handleIdle(schedule *snapschedulerv1alpha1.SnapshotSchedule,
	logger logr.Logger, c client.Client) (reconcile.Result, error) {
	timeNow := time.Now()
	timeNext := schedule.Status.NextSnapshotTime.Time

	if !schedule.Spec.Disabled && timeNow.After(timeNext) {
		// It's time to take snaps... switch state
		schedule.Status.State = snapschedulerv1alpha1.StateSnapshotting
		return reconcile.Result{}, nil
	}

	// We always update nextSnapshot in case the schedule changed
	if err := updateNextSnapTime(schedule, timeNow); err != nil {
		logger.Error(err, "couldn't update next snap time",
			"cronspec", schedule.Spec.Schedule)
		return reconcile.Result{}, err
	}

	if err := expireByTime(schedule, logger, c); err != nil {
		logger.Error(err, "expireByTime")
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

func handleSnapshotting(schedule *snapschedulerv1alpha1.SnapshotSchedule,
	logger logr.Logger, c client.Client) (reconcile.Result, error) {
	pvcList, err := listPVCsMatchingSelector(logger, c, schedule.Namespace, &schedule.Spec.ClaimSelector)
	if err != nil {
		logger.Error(err, "unable to get matching PVCs")
		return reconcile.Result{}, err
	}

	// Iterate through the PVCs and make sure snapshots exist for each. We
	// stop and re-queue at the first error.
	snapTime := schedule.Status.NextSnapshotTime.Time.UTC()
	for _, pvc := range pvcList.Items {
		snapName := snapshotName(pvc.Name, schedule.Name, snapTime)
		found := &snapv1alpha1.VolumeSnapshot{}
		logger.Info("looking for snapshot", "name", snapName)
		err = c.Get(context.TODO(), types.NamespacedName{Name: snapName, Namespace: pvc.Namespace}, found)
		if err != nil {
			if kerrors.IsNotFound(err) {
				logger.Info("creating a snapshot", "PVC", pvc.Name, "Snapshot", snapName)
				snap := newSnapForClaim(snapName, pvc, schedule.Name, snapTime,
					schedule.Spec.SnapshotTemplate.Labels,
					schedule.Spec.SnapshotTemplate.SnapshotClassName)
				err = c.Create(context.TODO(), &snap)
			} else {
				logger.Error(err, "looking for snapshot", "name", snapName)
				return reconcile.Result{}, err
			}
		}
		if err != nil {
			logger.Error(err, "while creating snapshots", "name", snapName)
			return reconcile.Result{}, err
		}
	}

	// We're done taking snapshots. Transition back to idle
	schedule.Status.State = snapschedulerv1alpha1.StateIdle
	// Update lastSnapshot & nextSnapshot times
	timeNow := metav1.Now()
	schedule.Status.LastSnapshotTime = &timeNow
	if err = updateNextSnapTime(schedule, timeNow.Time); err != nil {
		logger.Error(err, "couldn't update next snap time",
			"cronspec", schedule.Spec.Schedule)
		return reconcile.Result{}, err
	}
	// Changing .status will automatically cause requeuing
	return reconcile.Result{}, nil
}

func snapshotName(pvcName string, scheduleName string, time time.Time) string {
	return pvcName + "-" + scheduleName + "-" + time.Format(timeYYYYMMDDHHMMSS)
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
func newSnapForClaim(snapName string, pvc corev1.PersistentVolumeClaim,
	scheduleName string, scheduleTime time.Time,
	labels map[string]string, snapClass *string) snapv1alpha1.VolumeSnapshot {
	numLabels := 2
	if labels != nil {
		numLabels += len(labels)
	}
	snapLabels := make(map[string]string, numLabels)
	for k, v := range labels {
		snapLabels[k] = v
	}
	snapLabels[ScheduleKey] = scheduleName
	snapLabels[WhenKey] = scheduleTime.Format(timeYYYYMMDDHHMMSS)
	return snapv1alpha1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      snapName,
			Namespace: pvc.Namespace,
			Labels:    snapLabels,
		},
		Spec: snapv1alpha1.VolumeSnapshotSpec{
			Source: &corev1.TypedLocalObjectReference{
				APIGroup: nil,
				Kind:     "PersistentVolumeClaim",
				Name:     pvc.Name,
			},
			VolumeSnapshotClassName: snapClass,
		},
	}
}

// listPVCsMatchingSelector retrieves a list of PVCs that match the given selector
func listPVCsMatchingSelector(logger logr.Logger, c client.Client,
	namespace string, ls *metav1.LabelSelector) (*corev1.PersistentVolumeClaimList, error) {
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

func getExpirationTime(schedule *snapschedulerv1alpha1.SnapshotSchedule,
	now time.Time, logger logr.Logger) (*time.Time, error) {
	if schedule.Spec.Retention.Expires == "" {
		// No time-based retention configured
		return nil, nil
	}

	lifetime, err := time.ParseDuration(schedule.Spec.Retention.Expires)
	if err != nil {
		logger.Error(err, "unable to parse spec.retention.expires")
		return nil, err
	}

	if lifetime < 0 {
		err := errors.New("duration must be greater than 0")
		logger.Error(err, "invalid value for spec.retention.expires")
		return nil, err
	}

	expiration := now.Add(-lifetime).UTC()
	return &expiration, nil
}

func expireByTime(schedule *snapschedulerv1alpha1.SnapshotSchedule,
	logger logr.Logger, c client.Client) error {
	expiration, err := getExpirationTime(schedule, time.Now(), logger)
	if err != nil {
		logger.Error(err, "unable to determine snapshot expiration time")
		return err
	}
	if expiration == nil {
		// No time-based retention configured
		return nil
	}

	logger.Info("expiring snapshots", "expiration", expiration.Format(time.RFC3339))

	labelSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			ScheduleKey: schedule.Name,
		},
	}
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		logger.Error(err, "unable to create label selector for snapshot expiration")
		return err
	}

	snapList := &snapv1alpha1.VolumeSnapshotList{}
	err = c.List(context.TODO(),
		&client.ListOptions{LabelSelector: selector, Namespace: schedule.Namespace}, snapList)
	if err != nil {
		logger.Error(err, "unable to retrieve list of snapshots")
		return err
	}

	logger.Info("evaluating snapshots", "count", len(snapList.Items))
	for _, snap := range snapList.Items {
		if snap.CreationTimestamp.Time.Before(*expiration) {
			logger.Info("deleting expired snapshot", "name", snap.Name)
			err = c.Delete(context.TODO(), &snap, client.PropagationPolicy(metav1.DeletePropagationBackground))
			if err != nil {
				logger.Error(err, "error deleting snapshot", "name", snap.Name)
				return err
			}
		}
	}

	return nil
}
