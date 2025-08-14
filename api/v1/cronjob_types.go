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

package v1

import (
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CronJobSpec defines the desired state of CronJob.
type CronJobSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +required
	Profile JobProfileSpec `json:"jobProfile"`

	// Additional `cronjobs.v1.batch.metadata`.
	// +optional
	Metadata CronJobMetadata `json:"cronJobMetadata,omitempty"`

	// The schedule in Cron format, see https://en.wikipedia.org/wiki/Cron.
	//
	// +required
	Schedule string `json:"schedule"`

	// The time zone name for the given schedule, see https://en.wikipedia.org/wiki/List_of_tz_database_time_zones.
	// If not specified, this will default to the time zone of the kube-controller-manager process.
	// The set of valid time zone names and the time zone offset is loaded from the system-wide time zone
	// database by the API server during CronJob validation and the controller manager during execution.
	// If no system-wide time zone database can be found a bundled version of the database is used instead.
	// If the time zone name becomes invalid during the lifetime of a CronJob or due to a change in host
	// configuration, the controller will stop creating new new Jobs and will create a system event with the
	// reason UnknownTimeZone.
	// More information can be found in https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/#time-zones
	// +optional
	TimeZone *string `json:"timeZone,omitempty"`

	// Optional deadline in seconds for starting the job if it misses scheduled
	// time for any reason.  Missed jobs executions will be counted as failed ones.
	// +optional
	StartingDeadlineSeconds *int64 `json:"startingDeadlineSeconds,omitempty"`

	// Specifies how to treat concurrent executions of a Job.
	// Valid values are:
	//
	// - "Allow" (default): allows CronJobs to run concurrently;
	// - "Forbid": forbids concurrent runs, skipping next run if previous run hasn't finished yet;
	// - "Replace": cancels currently running job and replaces it with a new one
	// +kubebuilder:default:value=Allow
	// +optional
	ConcurrencyPolicy batchv1.ConcurrencyPolicy `json:"concurrencyPolicy,omitempty"`

	// This flag tells the controller to suspend subsequent executions, it does
	// not apply to already started executions. Defaults to false.
	// +kubebuilder:default:value=false
	// +optional
	Suspend *bool `json:"suspend,omitempty"`

	// The number of successful finished jobs to retain.
	// This is a pointer to distinguish between explicit zero and not specified.
	// +kubebuilder:default:value=3
	// +optional
	SuccessfulJobsHistoryLimit *int32 `json:"successfulJobsHistoryLimit,omitempty"`

	// The number of failed finished jobs to retain.
	// This is a pointer to distinguish between explicit zero and not specified.
	// +kubebuilder:default:value=1
	// +optional
	FailedJobsHistoryLimit *int32 `json:"failedJobsHistoryLimit,omitempty"`
}

type CronJobMetadata struct {
	// Additional labels for generated `cronjobs.v1.batch`.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
	// Additional annotations for generated `cronjobs.v1.batch`.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
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
	// Count of hash collisions for the Job. The Job controller uses this
	// field as a collision avoidance mechanism when it needs to create the name for the
	// newest `jobs.v1.batch`.
	// +optional
	CollisionCount *int32 `json:"collisionCount,omitempty"`
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
// +kubebuilder:printcolumn:name="Schedule",type="string",JSONPath=".spec.profile.cronJobParams.schedule"
// +kubebuilder:printcolumn:name="Timezone",type="string",JSONPath=".spec.profile.cronJobParams.timeZone"
// +kubebuilder:printcolumn:name="Suspend",type="boolean",JSONPath=".spec.profile.cronJobParams.suspend"
// +kubebuilder:printcolumn:name="Profile",type="string",JSONPath=".spec.profile.jobProfile.podProfileRef"
// +kubebuilder:printcolumn:name="Available",type="string",JSONPath=".status.conditions[?(@.type==\"Available\")].status"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[?(@.type==\"Available\")].reason"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[?(@.type==\"Available\")].message"

// CronJob represents a template of `cronjobs.v1.batch`.
//
// CronJob generates `cronjobs.v1.batch.spec` from `spec.jobProfile`,
// and then combines it with `spec.schedule`, `spec.timeZone`, ... to produce `cronjobs.v1.batch`.
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
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CronJobSpec   `json:"spec,omitempty"`
	Status CronJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CronJobList contains a list of CronJob.
type CronJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CronJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CronJob{}, &CronJobList{})
}
