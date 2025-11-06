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
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// JobProfileSpec defines the desired state of JobProfile
type JobProfileSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Template describes the jobs that will be created.
	// +required
	Template JobTemplate `json:"template"`
}

// JobProfileTemplate defines the jobs that will be created.
type JobTemplate struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// `jobs.v1.batch.spec` configuration options.
	// +optional
	JobParams JobParams `json:"jobParams"`
}

// `jobs.v1.batch.spec` without template, ttlSecondsAfterFinished.
type JobParams struct {
	// Specifies the maximum desired number of pods the job should
	// run at any given time. The actual number of pods running in steady state will
	// be less than this number when ((.spec.completions - .status.successful) < .spec.parallelism),
	// i.e. when the work left to do is less than max parallelism.
	// More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/
	// +optional
	Parallelism *int32 `json:"parallelism,omitempty"`

	// Specifies the desired number of successfully finished pods the
	// job should be run with.  Setting to null means that the success of any
	// pod signals the success of all pods, and allows parallelism to have any positive
	// value.  Setting to 1 means that parallelism is limited to 1 and the success of that
	// pod signals the success of the job.
	// More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/
	// +optional
	Completions *int32 `json:"completions,omitempty"`

	// Specifies the duration in seconds relative to the startTime that the job
	// may be continuously active before the system tries to terminate it; value
	// must be positive integer. If a Job is suspended (at creation or through an
	// update), this timer will effectively be stopped and reset when the Job is
	// resumed again.
	// +optional
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty"`

	// Specifies the policy of handling failed pods. In particular, it allows to
	// specify the set of actions and conditions which need to be
	// satisfied to take the associated action.
	// If empty, the default behaviour applies - the counter of failed pods,
	// represented by the jobs's .status.failed field, is incremented and it is
	// checked against the backoffLimit. This field cannot be used in combination
	// with restartPolicy=OnFailure.
	//
	// +optional
	PodFailurePolicy *batchv1.PodFailurePolicy `json:"podFailurePolicy,omitempty"`

	// successPolicy specifies the policy when the Job can be declared as succeeded.
	// If empty, the default behavior applies - the Job is declared as succeeded
	// only when the number of succeeded pods equals to the completions.
	// When the field is specified, it must be immutable and works only for the Indexed Jobs.
	// Once the Job meets the SuccessPolicy, the lingering pods are terminated.
	//
	// This field is beta-level. To use this field, you must enable the
	// `JobSuccessPolicy` feature gate (enabled by default).
	// +optional
	SuccessPolicy *batchv1.SuccessPolicy `json:"successPolicy,omitempty"`

	// Specifies the number of retries before marking this job failed.
	// Defaults to 6
	// +kubebuilder:default:value=6
	// +optional
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`

	// Specifies the limit for the number of retries within an
	// index before marking this index as failed. When enabled the number of
	// failures per index is kept in the pod's
	// batch.kubernetes.io/job-index-failure-count annotation. It can only
	// be set when Job's completionMode=Indexed, and the Pod's restart
	// policy is Never. The field is immutable.
	// This field is beta-level. It can be used when the `JobBackoffLimitPerIndex`
	// feature gate is enabled (enabled by default).
	// +optional
	BackoffLimitPerIndex *int32 `json:"backoffLimitPerIndex,omitempty"`

	// Specifies the maximal number of failed indexes before marking the Job as
	// failed, when backoffLimitPerIndex is set. Once the number of failed
	// indexes exceeds this number the entire Job is marked as Failed and its
	// execution is terminated. When left as null the job continues execution of
	// all of its indexes and is marked with the `Complete` Job condition.
	// It can only be specified when backoffLimitPerIndex is set.
	// It can be null or up to completions. It is required and must be
	// less than or equal to 10^4 when is completions greater than 10^5.
	// This field is beta-level. It can be used when the `JobBackoffLimitPerIndex`
	// feature gate is enabled (enabled by default).
	// +optional
	MaxFailedIndexes *int32 `json:"maxFailedIndexes,omitempty"`

	// TODO enabled it when https://github.com/kubernetes/kubernetes/issues/28486 has been fixed
	// Optional number of failed pods to retain.
	// +optional
	// FailedPodsLimit *int32 `json:"failedPodsLimit,omitempty" protobuf:"varint,9,opt,name=failedPodsLimit"`

	// A label query over pods that should match the pod count.
	// Normally, the system sets this field for you.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
	// +optional
	Selector *metav1.LabelSelector `json:"selector,omitempty"`

	// manualSelector controls generation of pod labels and pod selectors.
	// Leave `manualSelector` unset unless you are certain what you are doing.
	// When false or unset, the system pick labels unique to this job
	// and appends those labels to the pod template.  When true,
	// the user is responsible for picking unique labels and specifying
	// the selector.  Failure to pick a unique label may cause this
	// and other jobs to not function correctly.  However, You may see
	// `manualSelector=true` in jobs that were created with the old `extensions/v1beta1`
	// API.
	// More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/#specifying-your-own-pod-selector
	// +optional
	ManualSelector *bool `json:"manualSelector,omitempty"`

	// ttlSecondsAfterFinished limits the lifetime of a Job that has finished
	// execution (either Complete or Failed). If this field is set,
	// ttlSecondsAfterFinished after the Job finishes, it is eligible to be
	// automatically deleted. When the Job is being deleted, its lifecycle
	// guarantees (e.g. finalizers) will be honored. If this field is unset,
	// the Job won't be automatically deleted. If this field is set to zero,
	// the Job becomes eligible to be deleted immediately after it finishes.
	//
	// When CRD [Job] uses this, even if the TTL is expired, one [Job](https://kubernetes.io/ja/docs/concepts/workloads/controllers/job/) will remain. It will be reflected in
	// the `jobs.pixiv.net/ttl-seconds-after-finished` annotation instead of `jobs.v1.batch.spec.ttlSecondsAfterFinished`.
	// +optional
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`

	// completionMode specifies how Pod completions are tracked. It can be
	// `NonIndexed` (default) or `Indexed`.
	//
	// `NonIndexed` means that the Job is considered complete when there have
	// been .spec.completions successfully completed Pods. Each Pod completion is
	// homologous to each other.
	//
	// `Indexed` means that the Pods of a
	// Job get an associated completion index from 0 to (.spec.completions - 1),
	// available in the annotation batch.kubernetes.io/job-completion-index.
	// The Job is considered complete when there is one successfully completed Pod
	// for each index.
	// When value is `Indexed`, .spec.completions must be specified and
	// `.spec.parallelism` must be less than or equal to 10^5.
	// In addition, The Pod name takes the form
	// `$(job-name)-$(index)-$(random-string)`,
	// the Pod hostname takes the form `$(job-name)-$(index)`.
	//
	// More completion modes can be added in the future.
	// If the Job controller observes a mode that it doesn't recognize, which
	// is possible during upgrades due to version skew, the controller
	// skips updates for the Job.
	// +kubebuilder:default:value=NonIndexed
	// +optional
	CompletionMode *batchv1.CompletionMode `json:"completionMode,omitempty"`

	// suspend specifies whether the Job controller should create Pods or not. If
	// a Job is created with suspend set to true, no Pods are created by the Job
	// controller. If a Job is suspended after creation (i.e. the flag goes from
	// false to true), the Job controller will delete all active Pods associated
	// with this Job. Users must design their workload to gracefully handle this.
	// Suspending a Job will reset the StartTime field of the Job, effectively
	// resetting the ActiveDeadlineSeconds timer too. Defaults to false.
	//
	// +kubebuilder:default:value=false
	// +optional
	Suspend *bool `json:"suspend,omitempty"`

	// podReplacementPolicy specifies when to create replacement Pods.
	// Possible values are:
	// - TerminatingOrFailed means that we recreate pods
	//   when they are terminating (has a metadata.deletionTimestamp) or failed.
	// - Failed means to wait until a previously created Pod is fully terminated (has phase
	//   Failed or Succeeded) before creating a replacement Pod.
	//
	// When using podFailurePolicy, Failed is the the only allowed value.
	// TerminatingOrFailed and Failed are allowed values when podFailurePolicy is not in use.
	// This is an beta field. To use this, enable the JobPodReplacementPolicy feature toggle.
	// This is on by default.
	// +optional
	PodReplacementPolicy *batchv1.PodReplacementPolicy `json:"podReplacementPolicy,omitempty"`

	// ManagedBy field indicates the controller that manages a Job. The k8s Job
	// controller reconciles jobs which don't have this field at all or the field
	// value is the reserved string `kubernetes.io/job-controller`, but skips
	// reconciling Jobs with a custom value for this field.
	// The value must be a valid domain-prefixed path (e.g. acme.io/foo) -
	// all characters before the first "/" must be a valid subdomain as defined
	// by RFC 1123. All characters trailing the first "/" must be valid HTTP Path
	// characters as defined by RFC 3986. The value cannot exceed 63 characters.
	// This field is immutable.
	//
	// This field is alpha-level. The job controller accepts setting the field
	// when the feature gate JobManagedBy is enabled (disabled by default).
	// +optional
	ManagedBy *string `json:"managedBy,omitempty"`
}

// JobProfileStatus defines the observed state of JobProfile.
type JobProfileStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// JobProfile is the Schema for the jobprofiles API
type JobProfile struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of JobProfile
	// +required
	Spec JobProfileSpec `json:"spec"`

	// status defines the observed state of JobProfile
	// +optional
	Status JobProfileStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// JobProfileList contains a list of JobProfile
type JobProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JobProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JobProfile{}, &JobProfileList{})
}
