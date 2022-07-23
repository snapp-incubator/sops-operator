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

// SopsSecretTemplate defines the map of secrets to create
type SopsSecretTemplate struct {
	// Name of the Kubernetes secret to create
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Annotations to apply to Kubernetes secret
	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels to apply to Kubernetes secret
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`

	// Kubernetes secret type. Default: Opaque. Possible values: Opaque,
	// kubernetes.io/service-account-token, kubernetes.io/dockercfg,
	// kubernetes.io/dockerconfigjson, kubernetes.io/basic-auth,
	// kubernetes.io/ssh-auth, kubernetes.io/tls, bootstrap.kubernetes.io/token
	// +kubebuilder:validation:Optional
	Type string `json:"type,omitempty"`

	// Data map to use in Kubernetes secret (equivalent to Kubernetes Secret object data, please see for more
	// information: https://kubernetes.io/docs/concepts/configuration/secret/#overview-of-secrets)
	// +kubebuilder:validation:Optional
	Data map[string]string `json:"data,omitempty"`

	// stringData map to use in Kubernetes secret (equivalent to Kubernetes Secret object stringData, please see for more
	// information: https://kubernetes.io/docs/concepts/configuration/secret/#overview-of-secrets)
	// +kubebuilder:validation:Optional
	StringData map[string]string `json:"stringData,omitempty"`
}

// SopsSecretSpec defines the desired state of SopsSecret
type SopsSecretSpec struct {
	// +kubebuilder:validation:Required
	SecretTemplate SopsSecretTemplate `json:"secretTemplate,omitempty"`
	// +kubebuilder:validation:Required
	GPGKeyRefName string `json:"gpg_key_ref_name"`
	// +kubebuilder:validation:Optional
	Suspend bool `json:"suspend,omitempty"`
}

// SopsSecretStatus defines the observed state of SopsSecret
type SopsSecretStatus struct {
	// SopsSecret status message
	// +kubebuilder:validation:Optional
	Message string `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// SopsSecret is the Schema for the sopssecrets API
//+kubebuilder:resource:shortName=sops,scope=Namespaced
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.message`
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
