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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PodProfileSpec defines the desired state of PodProfile.
type PodProfileSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Template describes the pods that will be created.
	// +required
	Template PodProfileTemplate `json:"template"`
}

type PodProfileTemplate struct {
	// +optional
	PodProfileTemplateMetadata `json:"metadata,omitempty"`
	// Specification of the desired behavior of the pod.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +optional
	Spec corev1.PodSpec `json:"spec,omitempty"`
}

type PodProfileTemplateMetadata struct {
	// Additional labels for generated pod.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
	// Additional annotations for generated pod.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// PodProfileStatus defines the observed state of PodProfile.
type PodProfileStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PodProfile represents the configuration of a Pod.
type PodProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PodProfileSpec   `json:"spec,omitempty"`
	Status PodProfileStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PodProfileList contains a list of PodProfile.
type PodProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PodProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PodProfile{}, &PodProfileList{})
}
