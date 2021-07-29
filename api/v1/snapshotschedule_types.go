/*
Copyright (C) 2019  The snapscheduler authors

This file may be used, at your option, according to either the GNU AGPL 3.0 or
the Apache V2 license.

---
This program is free software: you can redistribute it and/or modify it under
the terms of the GNU Affero General Public License as published by the Free
Software Foundation, either version 3 of the License, or (at your option) any
later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY
WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A
PARTICULAR PURPOSE.  See the GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License along
with this program.  If not, see <https://www.gnu.org/licenses/>.

---
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

//nolint: lll
package v1

import (
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SnapshotRetentionSpec defines how long snapshots should be kept.
type SnapshotRetentionSpec struct {
	// The length of time (time.Duration) after which a given Snapshot will be
	// deleted.
	//+kubebuilder:validation:Pattern=^\d+(h|m|s)$
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Expiration period",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	//+optional
	Expires string `json:"expires,omitempty"`
	// The maximum number of snapshots to retain per PVC
	//+kubebuilder:validation:Minimum=1
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Maximum snapshots",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	//+optional
	MaxCount *int32 `json:"maxCount,omitempty"`
}

// SnapshotTemplateSpec defines the template for Snapshot objects
type SnapshotTemplateSpec struct {
	// A list of labels that should be added to each Snapshot created by this
	// schedule.
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	//+optional
	Labels map[string]string `json:"labels,omitempty"`
	// The name of the VolumeSnapshotClass to be used when creating Snapshots.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="VolumeSnapshotClass name",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	//+optional
	SnapshotClassName *string `json:"snapshotClassName,omitempty"`
}

// SnapshotScheduleSpec defines the desired state of SnapshotSchedule
type SnapshotScheduleSpec struct {
	// A filter to select which PVCs to snapshot via this schedule
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="PVC selector",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:selector:core:v1:PersistentVolumeClaim"}
	//+optional
	ClaimSelector metav1.LabelSelector `json:"claimSelector,omitempty"`
	// Retention determines how long this schedule's snapshots will be kept.
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	//+optional
	Retention SnapshotRetentionSpec `json:"retention,omitempty"`
	// Schedule is a Cronspec specifying when snapshots should be taken. See
	// https://en.wikipedia.org/wiki/Cron for a description of the format.
	//+kubebuilder:validation:Pattern=`^((\d+|\*)(/\d+)?(\s+(\d+|\*)(/\d+)?){4}|@(hourly|daily|weekly|monthly|yearly))$`
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Schedule",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	Schedule string `json:"schedule,omitempty"`
	// Indicates that this schedule should be temporarily disabled
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Disabled",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	//+optional
	Disabled bool `json:"disabled,omitempty"`
	// A template to customize the Snapshots.
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	SnapshotTemplate *SnapshotTemplateSpec `json:"snapshotTemplate,omitempty"`
}

// SnapshotScheduleStatus defines the observed state of SnapshotSchedule
type SnapshotScheduleStatus struct {
	// Conditions is a list of conditions related to operator reconciliation.
	//+optional
	//+operator-sdk:csv:customresourcedefinitions:type=status,displayName="Conditions",xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	Conditions []conditionsv1.Condition `json:"conditions,omitempty"`
	// The time of the most recent snapshot taken by this schedule
	//+optional
	//+operator-sdk:csv:customresourcedefinitions:type=status,displayName="Last snapshot",xDescriptors={"urn:alm:descriptor:text"}
	LastSnapshotTime *metav1.Time `json:"lastSnapshotTime,omitempty"`
	// The time of the next scheduled snapshot
	//+optional
	//+operator-sdk:csv:customresourcedefinitions:type=status,displayName="Next snapshot",xDescriptors={"urn:alm:descriptor:text"}
	NextSnapshotTime *metav1.Time `json:"nextSnapshotTime,omitempty"`
}

const (
	// ConditionReconciled is a Condition indicating whether the object is fully
	// reconciled.
	ConditionReconciled conditionsv1.ConditionType = "Reconciled"
	// ReconciledReasonError indicates there was an error while attempting to reconcile.
	ReconciledReasonError = "ReconcileError"
	// ReconciledReasonComplete indicates reconcile was successful
	ReconciledReasonComplete = "ReconcileComplete"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Schedule",type=string,JSONPath=".spec.schedule"
//+kubebuilder:printcolumn:name="Max age",type=string,JSONPath=".spec.retention.expires"
//+kubebuilder:printcolumn:name="Max num",type=integer,JSONPath=".spec.retention.maxCount"
//+kubebuilder:printcolumn:name="Disabled",type=boolean,JSONPath=".spec.disabled"
//+kubebuilder:printcolumn:name="Next snapshot",type=string,JSONPath=".status.nextSnapshotTime"
//+kubebuilder:resource:path=snapshotschedules,scope=Namespaced
//+operator-sdk:csv:customresourcedefinitions:displayName="Snapshot Schedule",resources={}

// SnapshotSchedule defines a schedule for taking automated snapshots of PVC(s)
type SnapshotSchedule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SnapshotScheduleSpec   `json:"spec,omitempty"`
	Status SnapshotScheduleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SnapshotScheduleList contains a list of SnapshotSchedule
type SnapshotScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SnapshotSchedule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SnapshotSchedule{}, &SnapshotScheduleList{})
}
