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

// SecretKeyConfig defines a single secret key and how to populate its value.
type SecretKeyConfig struct {
	// Value is the secret value. If empty and GenerateType is set, a value will be auto-generated.
	// +kubebuilder:validation:Optional
	Value string `json:"value,omitempty"`

	// GenerateType specifies how to auto-generate this key's value when Value is empty.
	// Supported types: "password" (alphanumeric), "hex" (hex-encoded), "uuid" (UUID v4), "base64" (base64-encoded random bytes).
	// If empty and Value is empty, the key will be created with an empty string.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=password;hex;uuid;base64;""
	GenerateType string `json:"generateType,omitempty"`

	// Length specifies the length of the generated value (default: 32).
	// For "hex" type, this is the number of random bytes (output will be 2x this length).
	// For "base64" type, this is the number of random bytes before encoding.
	// Ignored when Value is provided.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=8
	// +kubebuilder:validation:Maximum=128
	// +kubebuilder:default=32
	Length int `json:"length,omitempty"`
}

// VaultSecretRequestSpec defines the desired state of VaultSecretRequest.
type VaultSecretRequestSpec struct {
	// Mount is the Vault KV v2 mount point.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="kv"
	Mount string `json:"mount,omitempty"`

	// Path is the secret path within the mount (e.g., "domains", "core-services/authentik").
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Path string `json:"path"`

	// SecretKeys defines the key-value pairs to store in Vault at the specified path.
	// +kubebuilder:validation:Required
	SecretKeys map[string]SecretKeyConfig `json:"secretKeys"`

	// OverwriteExisting controls whether an existing Vault secret at this path
	// will be updated. If false and the secret already exists, no changes are made.
	// User-provided values (non-empty Value field) always update their respective keys
	// regardless of this setting when the secret exists.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	OverwriteExisting bool `json:"overwriteExisting,omitempty"`
}

// VaultSecretRequestStatus defines the observed state of VaultSecretRequest.
type VaultSecretRequestStatus struct {
	// Synced indicates whether the secret has been successfully written to Vault.
	Synced bool `json:"synced,omitempty"`

	// LastSyncedTime is the timestamp of the last successful sync to Vault.
	LastSyncedTime *metav1.Time `json:"lastSyncedTime,omitempty"`

	// Message provides additional information about the current state.
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Path",type=string,JSONPath=`.spec.path`
// +kubebuilder:printcolumn:name="Synced",type=boolean,JSONPath=`.status.synced`
// +kubebuilder:printcolumn:name="Last Synced",type=date,JSONPath=`.status.lastSyncedTime`

// VaultSecretRequest is the Schema for the vaultsecretrequests API.
// It defines a desired secret in HashiCorp Vault and ensures it exists with the specified keys.
type VaultSecretRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VaultSecretRequestSpec   `json:"spec,omitempty"`
	Status VaultSecretRequestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VaultSecretRequestList contains a list of VaultSecretRequest.
type VaultSecretRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VaultSecretRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VaultSecretRequest{}, &VaultSecretRequestList{})
}
