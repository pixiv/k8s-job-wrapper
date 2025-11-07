/*
Copyright 2025 pixiv Inc.

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

package v2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CronJobSpec defines the desired state of CronJob
type CronJobSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +required
	CronJobProfile CronJobProfileRef `json:"cronJobProfile"`
	// +required
	PodProfile PodProfileRef `json:"podProfile"`
	// +required
	JobProfile JobProfileRef `json:"jobProfile"`
}

type CronJobProfileRef struct {
	// Name of [CronJobProfile] that this refers to.
	// +required
	Ref string `json:"ref"`
	// Patches to be applied to `template` of [PodProfile].
	// +listType=atomic
	// +optional
	Patches []JobPatch `json:"patches,omitempty"`
}

// CronJobStatus defines the observed state of CronJob.
type CronJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Conditions represent the latest available observations of an object's state.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type CronJobConditionType string

const (
	// CronJobAvailable means the CronJob applied successfully.
	CronJobAvailable CronJobConditionType = "Available"
	// CronJobDegraded means the CronJob failed to apply.
	CronJobDegraded CronJobConditionType = "Degraded"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="PodProfile",type="string",JSONPath=".spec.podProfile.ref"
// +kubebuilder:printcolumn:name="JobProfile",type="string",JSONPath=".spec.jobProfile.ref"
// +kubebuilder:printcolumn:name="CronJobProfile",type="string",JSONPath=".spec.cronJobProfile.ref"
// +kubebuilder:printcolumn:name="Available",type="string",JSONPath=".status.conditions[?(@.type==\"Available\")].status"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[?(@.type==\"Available\")].reason"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[?(@.type==\"Available\")].message"

// CronJob represents a template of `cronjobs.v1.batch`.
//
// CronJob generates `cronjobs.v1.batch.spec` from the profiles specified in `spec.jobProfile`, `spec.podProfile`, `spec.cronJobProfile`.
//
// The generated `cronjobs.v1.batch` will always have the following labels applied:
//
//   - `app.kubernetes.io/created-by=pixiv-job-controller`
//   - `cronjobs.pixiv.net/name=$NAME_OF_CRONJOB`
//
// Where `$NAME_OF_CRONJOB` is the `metadata.name`.
//
// The generated `cronjobs.v1.batch` will have the suffix `-pxvcjob` appended to its name.
// The maximum length of `metadata.name` of `cronjobs.v1.batch` is 52 characters.
// Therefore, the maximum length of `metadata.name` of [CronJob] is 44 characters.
//
// +kubebuilder:validation:XValidation:rule="self.metadata.name.size() <= 44",message="must be no more than 44 characters"
type CronJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`
	Spec              CronJobSpec   `json:"spec"`
	Status            CronJobStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// CronJobList contains a list of CronJob
type CronJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CronJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CronJob{}, &CronJobList{})
}
