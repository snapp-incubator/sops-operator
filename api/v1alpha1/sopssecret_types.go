/*
Copyright 2022.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// SopsSecretManagedAnnotation is the name for the annotation for
	// flagging the existing secret be managed by SopsSecret controller.
	SopsSecretManagedAnnotation = "gitops-controller.snappcloud.io/managed"
)

// SopsSecretSpec defines the desired state of SopsSecret
type SopsSecretSpec struct {
	// +kubebuilder:validation:Required
	StringData map[string]string `json:"stringData,omitempty"`
	// +kubebuilder:validation:Required
	GPGKeyRefName string `json:"gpg_key_ref_name"`
	// +kubebuilder:validation:Optional
	Type string `json:"type,omitempty"`
	// +kubebuilder:validation:Optional
	Suspend bool `json:"suspend,omitempty"`
}

// SopsSecretStatus defines the observed state of SopsSecret
type SopsSecretStatus struct {
	// SopsSecret status message
	// +kubebuilder:validation:Optional
	Health  string `json:"health"`
	Message string `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// SopsSecret is the Schema for the sopssecrets API
//+kubebuilder:resource:shortName=sops,scope=Namespaced
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="Health",type=string,JSONPath=`.status.health`
//+kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.message`
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type SopsSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SopsSecretSpec   `json:"spec,omitempty"`
	Status SopsSecretStatus `json:"status,omitempty"`
	Sops   SopsMetadata     `json:"sops,omitempty"`
}

//+kubebuilder:object:root=true

// SopsSecretList contains a list of SopsSecret
type SopsSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SopsSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SopsSecret{}, &SopsSecretList{})
}
