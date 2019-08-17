package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

// +k8s:openapi-gen=true

// SnapshotRetentionSpec defines how long snapshots should be kept.
type SnapshotRetentionSpec struct {
	// Expires is the length of time (time.Duration) after which a given
	// Snapshot will be deleted.
	// +kubebuilder:validation:Pattern=^\d+(h|m|s)$
	Expires string `json:"expires,omitempty"`
	// +kubebuilder:validation:Minimum=1
	MaxCount *int32 `json:"maxCount,omitempty"`
}

// +k8s:openapi-gen=true

// SnapshotTemplateHookSpec defines the execution hooks that should be invoked
// before and after taking a snapshot.
type SnapshotTemplateHookSpec struct {
	PreSnapshot  string `json:"preSnapshot,omitempty"`
	PostSnapshot string `json:"postSnapshot,omitempty"`
}

// +k8s:openapi-gen=true

// SnapshotTemplateSpec defines the template for Snapshot objects
type SnapshotTemplateSpec struct {
	Labels map[string]string        `json:"labels,omitempty"`
	Hooks  SnapshotTemplateHookSpec `json:"hooks,omitempty"`
}

// +k8s:openapi-gen=true

// SnapshotScheduleSpec defines the desired state of SnapshotSchedule
type SnapshotScheduleSpec struct {
	// ClaimSelector selects which PVCs will be snapshotted according to
	// this schedule.
	ClaimSelector metav1.LabelSelector  `json:"claimSelector,omitempty"`
	Retention     SnapshotRetentionSpec `json:"retention,omitempty"`
	// +kubebuilder:validation:Pattern=^(\d+|\*)(/\d+)?(\s+(\d+|\*)(/\d+)?){4}$
	Schedule string `json:"schedule,omitempty"`
	// +kubebuilder:validation:Minimum=1
	StartDeadlineSeconds *int32               `json:"startDeadlineSeconds,omitempty"`
	Active               *bool                `json:"active,omitempty"`
	Template             SnapshotTemplateSpec `json:"template,omitempty"`
}

// +k8s:openapi-gen=true

// SnapshotScheduleStatus defines the observed state of SnapshotSchedule
type SnapshotScheduleStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status

// SnapshotSchedule is the Schema for the snapshotschedules API
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
