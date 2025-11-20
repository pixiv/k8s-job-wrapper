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

	// +required
	PodProfile PodProfileRef `json:"podProfile"`
	// +required
	JobProfile JobProfileRef `json:"jobProfile"`
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

type JobConditionType string

const (
	// JobAvailable means the Job applied successfully.
	JobAvailable JobConditionType = "Available"
	// JobDegraded means the Job failed to apply.
	JobDegraded JobConditionType = "Degraded"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +versionName=v2
// +kubebuilder:printcolumn:name="PodProfile",type="string",JSONPath=".spec.podProfile.ref"
// +kubebuilder:printcolumn:name="JobProfile",type="string",JSONPath=".spec.jobProfile.ref"
// +kubebuilder:printcolumn:name="Available",type="string",JSONPath=".status.conditions[?(@.type==\"Available\")].status"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[?(@.type==\"Available\")].reason"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[?(@.type==\"Available\")].message"

// Job represents a template of `jobs.v1.batch`.
//
// The Job creates a `jobs.v1.batch` manifest from the [PodProfile] specified in `spec.podProfile.ref` and [JobProfile] specified in `spec.jobProfile.ref`, and applies patches specified in `spec.podProfile.patches` and `spec.jobProfile.patches`.
// The generated `jobs.v1.batch` will always have the following labels:
//
//   - `app.kubernetes.io/created-by=pixiv-job-controller`
//   - `jobs.pixiv.net/name=$NAME_OF_JOB`
//   - `jobs.pixiv.net/job-spec-hash=$HASH_OF_JOB_SPEC`
//
// `$NAME_OF_JOB` is the value of `metadata.name`.
// `$HASH_OF_JOB_SPEC` is the hash value of `jobs.v1.batch.spec`.
//
// The Job generates a `jobs.v1.batch` in the following cases:
//
//   - When it generates the `jobs.v1.batch` for the first time.
//   - When it generates a manifest with a different spec called $HASH_OF_JOB_SPEC from the most recently generated jobs.v1.batch, and it has completed.
//
// The name of the `jobs.v1.batch` to be generated will have an 11-character suffix appended to `metadata.name`.
// Therefore, `metadata.name` can be a maximum of 52 characters long.
//
// +kubebuilder:validation:XValidation:rule="self.metadata.name.size() <= 52",message="must be no more than 52 characters"
type Job struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`
	Spec              JobSpec   `json:"spec"`
	Status            JobStatus `json:"status,omitempty,omitzero"`
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
