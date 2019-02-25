package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SnapshotScheduleSpec defines the desired state of SnapshotSchedule
// +k8s:openapi-gen=true
type SnapshotScheduleSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	// +kubebuilder:validation:Pattern=^(\d+|\*)(/\d+)?(\s+(\d+|\*)(/\d+)?){4}$
	Schedule  string                `json:"schedule,omitempty"`
	Retention SnapshotRetentionSpec `json:"retention,omitempty"`
}

// SnapshotRetentionSpec defines the retention policy for snapshots
// +k8s:openapi-gen=true
type SnapshotRetentionSpec struct {
	// +kubebuilder:validation:Minimum=1
	MaxCount *int32 `json:"maxCount,omitempty"`
}

// SnapshotScheduleStatus defines the observed state of SnapshotSchedule
// +k8s:openapi-gen=true
type SnapshotScheduleStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	ScheduledJob JobRef `json:"scheduledJob,omitempty"`
}

// JobRef is the namespace/name of the Job that implements this schedule
// +k8s:openapi-gen=true
type JobRef struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SnapshotSchedule is the Schema for the snapshotschedules API
// +k8s:openapi-gen=true
type SnapshotSchedule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SnapshotScheduleSpec   `json:"spec,omitempty"`
	Status SnapshotScheduleStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SnapshotScheduleList contains a list of SnapshotSchedule
type SnapshotScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SnapshotSchedule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SnapshotSchedule{}, &SnapshotScheduleList{})
}
