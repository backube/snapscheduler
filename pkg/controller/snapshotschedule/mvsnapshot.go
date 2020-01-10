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

package snapshotschedule

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	snapv1alpha1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	snapv1beta1 "github.com/kubernetes-csi/external-snapshotter/v2/pkg/apis/volumesnapshot/v1beta1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SnapshotVersionChecker keeps track of which VolumeSnapshot versions are valid in the API server.
type SnapshotVersionChecker struct {
	v1Beta1  bool
	v1Alpha1 bool
	client   *kubernetes.Clientset
}

// SetConfig sets the configuration the version checker should use to communicate w/ the API server.
func (svc *SnapshotVersionChecker) SetConfig(c *rest.Config) error {
	cs, err := kubernetes.NewForConfig(c)
	svc.client = cs
	return err
}

// Refresh queries the API server to determine the current valid versions.
func (svc *SnapshotVersionChecker) Refresh(logger logr.Logger) error {
	logger.V(4).Info("checking for VolumeSnapshot versions")
	dif := svc.client.Discovery()
	svc.v1Alpha1 = false
	rlist, err := dif.ServerResourcesForGroupVersion("snapshot.storage.k8s.io/v1alpha1")
	if err == nil {
		for _, resource := range rlist.APIResources {
			if resource.Name == "volumesnapshots" {
				logger.V(4).Info("VolumeSnapshots v1alpha1 found")
				svc.v1Alpha1 = true
			}
		}
	}

	svc.v1Beta1 = false
	rlist, err = dif.ServerResourcesForGroupVersion("snapshot.storage.k8s.io/v1beta1")
	if err == nil {
		for _, resource := range rlist.APIResources {
			if resource.Name == "volumesnapshots" {
				logger.V(4).Info("VolumeSnapshots v1beta1 found")
				svc.v1Beta1 = true
			}
		}
	}
	return nil
}

// VersionChecker is the cache of valid VolumeSnapshot versions.
var VersionChecker SnapshotVersionChecker

// MultiversionSnapshot abstracts the specific version of the VolumeSnapshot object.
type MultiversionSnapshot struct {
	v1Beta1  *snapv1beta1.VolumeSnapshot
	v1Alpha1 *snapv1alpha1.VolumeSnapshot
}

// ObjectMeta returns the metadata portion from the underlying VolumeSnapshot.
func (mvs *MultiversionSnapshot) ObjectMeta() *metav1.ObjectMeta {
	if mvs.v1Beta1 != nil {
		return &mvs.v1Beta1.ObjectMeta
	} else if mvs.v1Alpha1 != nil {
		return &mvs.v1Alpha1.ObjectMeta
	}
	return nil
}

// SourcePvcName returns the name of the PVC used as the source for the VolumeSnapshot.
func (mvs *MultiversionSnapshot) SourcePvcName() *string {
	if mvs.v1Beta1 != nil {
		return mvs.v1Beta1.Spec.Source.PersistentVolumeClaimName
	} else if mvs.v1Alpha1 != nil && mvs.v1Alpha1.Spec.Source != nil {
		return &mvs.v1Alpha1.Spec.Source.Name
	}
	return nil
}

// WrapSnapshotAlpha encapsulates a v1alpha1 VolumeSnapshot.
func WrapSnapshotAlpha(snap *snapv1alpha1.VolumeSnapshot) *MultiversionSnapshot {
	if snap == nil {
		return nil
	}
	return &MultiversionSnapshot{
		v1Alpha1: snap,
	}
}

// WrapSnapshotBeta encapsulates a v1beta1 VolumeSnapshot.
func WrapSnapshotBeta(snap *snapv1beta1.VolumeSnapshot) *MultiversionSnapshot {
	if snap == nil {
		return nil
	}
	return &MultiversionSnapshot{
		v1Beta1: snap,
	}
}

// GetMVSnapshot retrieves the VolumeSnapshot with the provided key from the API server.
// We use this function because the standard client.Get cannot be used with an mvs.
func GetMVSnapshot(ctx context.Context, client client.Reader, key types.NamespacedName) (*MultiversionSnapshot, error) {
	if VersionChecker.v1Beta1 {
		found := &snapv1beta1.VolumeSnapshot{}
		err := client.Get(ctx, key, found)
		if err == nil {
			mvs := &MultiversionSnapshot{
				v1Beta1: found,
			}
			return mvs, nil
		} else if !kerrors.IsNotFound(err) {
			return nil, err
		}
	}
	if VersionChecker.v1Alpha1 {
		found := &snapv1alpha1.VolumeSnapshot{}
		err := client.Get(ctx, key, found)
		if err == nil {
			mvs := &MultiversionSnapshot{
				v1Alpha1: found,
			}
			return mvs, nil
		}
	}
	return nil, kerrors.NewNotFound(schema.GroupResource{
		Group:    "snapshot.storage.k8s.io",
		Resource: "volumesnapshots",
	}, key.Name)
}

// ListMVSnapshot takes the place of client.List for MultiversionSnapshot objects.
func ListMVSnapshot(ctx context.Context,
	client client.Reader,
	opts ...client.ListOption) ([]MultiversionSnapshot, error) {
	outList := []MultiversionSnapshot{}
	if VersionChecker.v1Beta1 {
		betaList := &snapv1beta1.VolumeSnapshotList{}
		err := client.List(ctx, betaList, opts...)
		if err == nil {
			for _, snap := range betaList.Items {
				outList = append(outList, MultiversionSnapshot{
					v1Beta1: snap.DeepCopy(),
				})
			}
		} else {
			return []MultiversionSnapshot{}, err
		}
	}
	if VersionChecker.v1Alpha1 {
		alphaList := &snapv1alpha1.VolumeSnapshotList{}
		err := client.List(ctx, alphaList, opts...)
		if err == nil {
			for _, snap := range alphaList.Items {
				outList = append(outList, MultiversionSnapshot{
					v1Alpha1: snap.DeepCopy(),
				})
			}
		} else {
			return []MultiversionSnapshot{}, err
		}
	}
	return outList, nil
}

// Create saves the snapshot in the Kubernetes cluster.
func (mvs *MultiversionSnapshot) Create(ctx context.Context, client client.Writer, opts ...client.CreateOption) error {
	if mvs.v1Beta1 != nil {
		return client.Create(ctx, mvs.v1Beta1, opts...)
	}
	if mvs.v1Alpha1 != nil {
		return client.Create(ctx, mvs.v1Alpha1, opts...)
	}
	return nil
}

// Delete deletes the snapshot from Kubernetes cluster.
func (mvs *MultiversionSnapshot) Delete(ctx context.Context, client client.Writer, opts ...client.DeleteOption) error {
	if mvs.v1Beta1 != nil {
		return client.Delete(ctx, mvs.v1Beta1, opts...)
	}
	if mvs.v1Alpha1 != nil {
		return client.Delete(ctx, mvs.v1Alpha1, opts...)
	}
	return nil
}

// newSnapForClaim returns a VolumeSnapshot object based on a PVC
func newSnapForClaim(snapName string, pvc corev1.PersistentVolumeClaim,
	scheduleName string, scheduleTime time.Time,
	labels map[string]string, snapClass *string) *MultiversionSnapshot {
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
	mvs := &MultiversionSnapshot{}
	if VersionChecker.v1Beta1 {
		mvs.v1Beta1 = &snapv1beta1.VolumeSnapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      snapName,
				Namespace: pvc.Namespace,
				Labels:    snapLabels,
			},
			Spec: snapv1beta1.VolumeSnapshotSpec{
				Source: snapv1beta1.VolumeSnapshotSource{
					PersistentVolumeClaimName: &pvc.Name,
				},
				VolumeSnapshotClassName: snapClass,
			},
		}
		return mvs
	} else if VersionChecker.v1Alpha1 {
		mvs.v1Alpha1 = &snapv1alpha1.VolumeSnapshot{
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
		return mvs
	}
	return nil
}
