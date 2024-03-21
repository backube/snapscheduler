/*
Copyright 2021 The snapscheduler authors.

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

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	snapv1 "github.com/kubernetes-csi/external-snapshotter/client/v7/apis/volumesnapshot/v1"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	"github.com/robfig/cron/v3"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	snapschedulerv1 "github.com/backube/snapscheduler/api/v1"
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

// SnapshotScheduleReconciler reconciles a SnapshotSchedule object
type SnapshotScheduleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//nolint:lll
//+kubebuilder:rbac:groups=snapscheduler.backube,resources=snapshotschedules,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=snapscheduler.backube,resources=snapshotschedules/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=snapscheduler.backube,resources=snapshotschedules/finalizers,verbs=update
//+kubebuilder:rbac:groups=snapshot.storage.k8s.io,resources=volumesnapshots,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch

func (r *SnapshotScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx).WithValues("snapshotschedule", req.NamespacedName)
	reqLogger.Info("Reconciling SnapshotSchedule")

	// Fetch the SnapshotSchedule instance
	instance := &snapschedulerv1.SnapshotSchedule{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if kerrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	result, err := doReconcile(ctx, instance, reqLogger, r.Client)

	// Update result in CR
	if err != nil {
		conditionsv1.SetStatusCondition(&instance.Status.Conditions, conditionsv1.Condition{
			Type:    snapschedulerv1.ConditionReconciled,
			Status:  corev1.ConditionFalse,
			Reason:  snapschedulerv1.ReconciledReasonError,
			Message: err.Error(),
		})
	} else {
		conditionsv1.SetStatusCondition(&instance.Status.Conditions, conditionsv1.Condition{
			Type:    snapschedulerv1.ConditionReconciled,
			Status:  corev1.ConditionTrue,
			Reason:  snapschedulerv1.ReconciledReasonComplete,
			Message: "Reconcile complete",
		})
	}

	// Update instance.Status
	err2 := r.Client.Status().Update(ctx, instance)
	if err == nil { // Don't mask previous error
		err = err2
	}
	return result, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *SnapshotScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&snapschedulerv1.SnapshotSchedule{}).
		Complete(r)
}

func doReconcile(ctx context.Context, schedule *snapschedulerv1.SnapshotSchedule,
	logger logr.Logger, c client.Client) (ctrl.Result, error) {
	// If necessary, initialize time of next snap based on schedule
	if schedule.Status.NextSnapshotTime.IsZero() {
		// Update nextSnapshot time based on current time and cronspec
		if err := updateNextSnapTime(schedule, time.Now()); err != nil {
			logger.Error(err, "couldn't update next snap time",
				"cronspec", schedule.Spec.Schedule)
			return ctrl.Result{}, err
		}
	}

	timeNow := time.Now()
	timeNext := schedule.Status.NextSnapshotTime.Time
	if !schedule.Spec.Disabled && timeNow.After(timeNext) {
		// It's not necessary to check and contitionally return on error since
		// modifying .status will immediately cause an addl reconcile pass
		// (which will cover the rest of this reconcile function). We also don't
		// want to update nextSnapshot until this round is done.
		return handleSnapshotting(ctx, schedule, logger, c)
	}

	// We always update nextSnapshot in case the schedule changed
	if err := updateNextSnapTime(schedule, timeNow); err != nil {
		logger.Error(err, "couldn't update next snap time",
			"cronspec", schedule.Spec.Schedule)
		return ctrl.Result{}, err
	}

	if err := expireByTime(ctx, schedule, time.Now(), logger, c); err != nil {
		logger.Error(err, "expireByTime")
		return ctrl.Result{}, err
	}

	if err := expireByCount(ctx, schedule, logger, c); err != nil {
		logger.Error(err, "expireByCount")
		return ctrl.Result{}, err
	}
	// Ensure we requeue in time for the next scheduled snapshot time
	durTillNext := timeNext.Sub(timeNow)
	requeueTime := maxRequeueTime
	if durTillNext < requeueTime {
		requeueTime = durTillNext
	}
	return ctrl.Result{RequeueAfter: requeueTime}, nil
}

func handleSnapshotting(ctx context.Context, schedule *snapschedulerv1.SnapshotSchedule,
	logger logr.Logger, c client.Client) (ctrl.Result, error) {
	pvcList, err := listPVCsMatchingSelector(ctx, logger, c, schedule.Namespace, &schedule.Spec.ClaimSelector)
	if err != nil {
		logger.Error(err, "unable to get matching PVCs")
		return ctrl.Result{}, err
	}

	// Iterate through the PVCs and make sure snapshots exist for each. We
	// stop and re-queue at the first error.
	snapTime := schedule.Status.NextSnapshotTime.Time.UTC()
	for _, pvc := range pvcList.Items {
		snapName := snapshotName(pvc.Name, schedule.Name, snapTime)
		logger.V(4).Info("looking for snapshot", "name", snapName)
		key := types.NamespacedName{Name: snapName, Namespace: pvc.Namespace}
		snap := snapv1.VolumeSnapshot{}
		if err := c.Get(ctx, key, &snap); err != nil {
			if kerrors.IsNotFound(err) {
				labels := make(map[string]string)
				var snapshotClassName *string
				if schedule.Spec.SnapshotTemplate != nil {
					labels = schedule.Spec.SnapshotTemplate.Labels
					snapshotClassName = schedule.Spec.SnapshotTemplate.SnapshotClassName
				}
				snap := newSnapForClaim(snapName, pvc, schedule.Name, snapTime, labels, snapshotClassName)
				if snap != nil {
					logger.Info("creating a snapshot", "PVC", pvc.Name, "Snapshot", snapName)
					if err = c.Create(ctx, snap); err != nil {
						logger.Error(err, "while creating snapshots", "name", snapName)
						return ctrl.Result{}, err
					}
				} else {
					logger.Info("unable to create snapshot -- no supported VolumeSnapshot CRD is registered")
				}
			} else {
				logger.Error(err, "looking for snapshot", "name", snapName)
				return ctrl.Result{}, err
			}
		}
	}

	// Update lastSnapshot & nextSnapshot times
	timeNow := metav1.Now()
	schedule.Status.LastSnapshotTime = &timeNow
	if err = updateNextSnapTime(schedule, timeNow.Time); err != nil {
		logger.Error(err, "couldn't update next snap time",
			"cronspec", schedule.Spec.Schedule)
		return ctrl.Result{}, err
	}
	// Changing .status will automatically cause requeuing
	return ctrl.Result{}, nil
}

func snapshotName(pvcName string, scheduleName string, time time.Time) string {
	// How much room we have for PVC + schedule names
	nameBudget := validation.DNS1123SubdomainMaxLength - len(timeYYYYMMDDHHMMSS) - 2
	// Goal is to minimize the truncation. If one name is short, let the other use the excess
	if len(pvcName)+len(scheduleName) > nameBudget {
		pvcOverBudget := len(pvcName) > nameBudget/2
		scheduleOverBudget := len(scheduleName) > nameBudget/2
		if pvcOverBudget && !scheduleOverBudget {
			pvcName = pvcName[0 : nameBudget-len(scheduleName)]
		}
		if !pvcOverBudget && scheduleOverBudget {
			scheduleName = scheduleName[0 : nameBudget-len(pvcName)]
		}
		if pvcOverBudget && scheduleOverBudget {
			scheduleName = scheduleName[0 : nameBudget/2]
			pvcName = pvcName[0 : nameBudget/2]
		}
	}
	return pvcName + "-" + scheduleName + "-" + time.Format(timeYYYYMMDDHHMMSS)
}

func updateNextSnapTime(snapshotSchedule *snapschedulerv1.SnapshotSchedule, referenceTime time.Time) error {
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

// listPVCsMatchingSelector retrieves a list of PVCs that match the given selector
func listPVCsMatchingSelector(ctx context.Context, logger logr.Logger, c client.Client,
	namespace string, ls *metav1.LabelSelector) (*corev1.PersistentVolumeClaimList, error) {
	selector, err := metav1.LabelSelectorAsSelector(ls)
	if err != nil {
		return nil, err
	}
	pvcList := &corev1.PersistentVolumeClaimList{}
	listOpts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabelsSelector{
			Selector: selector,
		},
	}
	err = c.List(ctx, pvcList, listOpts...)
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

func newSnapForClaim(snapName string, pvc corev1.PersistentVolumeClaim,
	scheduleName string, scheduleTime time.Time,
	labels map[string]string, snapClass *string) *snapv1.VolumeSnapshot {
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
	return &snapv1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      snapName,
			Namespace: pvc.Namespace,
			Labels:    snapLabels,
		},
		Spec: snapv1.VolumeSnapshotSpec{
			Source: snapv1.VolumeSnapshotSource{
				PersistentVolumeClaimName: &pvc.Name,
			},
			VolumeSnapshotClassName: snapClass,
		},
	}
}
