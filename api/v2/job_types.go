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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// JobSpec defines the desired state of Job
type JobSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Additional `cronjob.v1.batch.metadata`. Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// +required
	PodProfile PodProfileRef `json:"podProfile"`
	// +required
	JobProfile JobProfileRef `json:"jobProfile"`

	// JobsHistoryLimit is the number of jobs to retain.
	// Value must be non-negative integer.
	// Default is 3.
	// +kubebuilder:default:value=3
	// +optional
	JobsHistoryLimit *int `json:"jobsHistoryLimit,omitempty"`
}

type PodProfileRef struct {
	// Name of [PodProfile] that this refers to.
	// +required
	Ref string `json:"ref"`
	// Patches to be applied to `template` of [PodProfile].
	// +listType=atomic
	// +optional
	Patches []JobPatch `json:"patches,omitempty"`
}

type JobProfileRef struct {
	// Name of [JobProfile] that this refers to.
	// +required
	Ref string `json:"ref"`
	// Patches to be applied to `template` of [JobProfile].
	// +listType=atomic
	// +optional
	Patches []JobPatch `json:"patches,omitempty"`
}

/*
JobPatch defines the [patch] to be applied to template of PodProfile.
For examples, rename container:
<pre><code>op: replace
path: /spec/containers/0/name
value: simple
</code></pre>

Replace command:
<pre><code>op: replace
path: /spec/containers/0/command
value: ["perl", "-Mbignum=bpi", "-wle", "print bpi(100)"]
</code></pre>

[patch]: https://datatracker.ietf.org/doc/html/rfc6902
*/
type JobPatch struct {
	// Operation to perform.
	// +required
	Operation string `json:"op"`
	// JSON-pointer that references a location within the template where the operation is performed.
	// The root of path will be `jobs.v1.batch.spec.template`.
	// +required
	Path string `json:"path"`
	// Any yaml object.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Value apiextensionsv1.JSON `json:"value,omitempty"`
	// +optional
	From string `json:"from,omitempty"`
}

// JobStatus defines the observed state of Job.
type JobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the Job resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Job is the Schema for the jobs API
type Job struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of Job
	// +required
	Spec JobSpec `json:"spec"`

	// status defines the observed state of Job
	// +optional
	Status JobStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// JobList contains a list of Job
type JobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Job `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Job{}, &JobList{})
}
