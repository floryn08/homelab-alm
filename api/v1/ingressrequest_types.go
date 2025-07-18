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

// IngressRequestSpec defines the desired state of IngressRequest.
type IngressRequestSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	VaultPath   string `json:"vaultPath"`
	Subdomain   string `json:"subdomain"`
	ServiceName string `json:"serviceName"`
	ServicePort string `json:"servicePort"`
	DomainKey   string `json:"domainKey"`

	Entrypoints []string `json:"entrypoints,omitempty"`

	TLS *IngressTLSConfig `json:"tls,omitempty"`

	Middlewares []MiddlewareRef `json:"middlewares,omitempty"`
}

type IngressTLSConfig struct {
	SecretName   string `json:"secretName,omitempty"`   // reference to TLS secret
	CertResolver string `json:"certResolver,omitempty"` // for dynamic certs (e.g. Let's Encrypt via Traefik)
}

type MiddlewareRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// IngressRequestStatus defines the observed state of IngressRequest.
type IngressRequestStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	FQDN string `json:"fqdn,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// IngressRequest is the Schema for the ingressrequests API.
type IngressRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IngressRequestSpec   `json:"spec,omitempty"`
	Status IngressRequestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IngressRequestList contains a list of IngressRequest.
type IngressRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IngressRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IngressRequest{}, &IngressRequestList{})
}
