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
	// Vault path to read domain configuration from
	// +kubebuilder:default="kv/data/domains"
	VaultPath string `json:"vaultPath,omitempty"`

	// The subdomain to prepend to the domain
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	Subdomain string `json:"subdomain"`

	// The name of the Kubernetes service to route traffic to
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	ServiceName string `json:"serviceName"`

	// The port of the service (can be port number or name)
	// +kubebuilder:validation:Required
	ServicePort string `json:"servicePort"`

	// The key used to fetch the domain from Vault
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	DomainKey string `json:"domainKey"`

	// Traefik entrypoints to use (defaults to ["web"])
	// +kubebuilder:validation:Optional
	Entrypoints []string `json:"entrypoints,omitempty"`

	// TLS configuration for the ingress
	// +kubebuilder:validation:Optional
	TLS *IngressTLSConfig `json:"tls,omitempty"`

	// Middlewares to apply to the route
	// +kubebuilder:validation:Optional
	Middlewares []MiddlewareRef `json:"middlewares,omitempty"`
}

type IngressTLSConfig struct {
	// Reference to TLS secret containing the certificate
	// +kubebuilder:validation:Optional
	SecretName string `json:"secretName,omitempty"`

	// CertResolver for dynamic certificates (e.g. Let's Encrypt via Traefik)
	// +kubebuilder:validation:Optional
	CertResolver string `json:"certResolver,omitempty"`
}

type MiddlewareRef struct {
	// Name of the Traefik middleware
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Namespace where the middleware is located
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
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
