/*
Copyright 2025.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CertificateRequestSpec defines the desired state of CertificateRequest.
type CertificateRequestSpec struct {
	// The name of the Kubernetes secret to store the generated certificate
	SecretName string `json:"secretName"`

	// The key used to fetch the domain from Vault at kv/data/domains
	DomainKey string `json:"domainKey"`

	// The subdomain to prepend to the domain (optional)
	Subdomain string `json:"subdomain,omitempty"`

	// Vault path (e.g. kv/data/cert-info) to read additional metadata (optional)
	VaultPath string `json:"vaultPath,omitempty"`
}

// CertificateRequestStatus defines the observed state of CertificateRequest.
type CertificateRequestStatus struct {
	// The computed fully qualified domain name (FQDN)
	FQDN string `json:"fqdn,omitempty"`

	// True if the Certificate has been successfully created
	Ready bool `json:"ready,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// CertificateRequest is the Schema for the certificaterequests API.
type CertificateRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CertificateRequestSpec   `json:"spec,omitempty"`
	Status CertificateRequestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CertificateRequestList contains a list of CertificateRequest.
type CertificateRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CertificateRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CertificateRequest{}, &CertificateRequestList{})
}
