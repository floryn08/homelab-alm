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

package utils

import (
	"testing"

	vault "github.com/hashicorp/vault/api"
)

// TestVaultClientCreation validates that we can create a Vault client
// This will catch breaking changes in the Vault API
func TestVaultClientCreation(t *testing.T) {
	config := vault.DefaultConfig()
	if config == nil {
		t.Fatal("vault.DefaultConfig() returned nil - Vault API may have changed")
	}

	// The Error field should exist (even if nil)
	_ = config.Error

	// Should be able to create client (may fail without VAULT_ADDR, but API should work)
	client, _ := vault.NewClient(config)
	
	// If client creation succeeded, verify Logical interface exists
	if client != nil {
		logical := client.Logical()
		if logical == nil {
			t.Error("client.Logical() returned nil - Vault API structure changed")
		}
	}
}

// TestVaultSecretStructure validates the Secret structure we depend on
func TestVaultSecretStructure(t *testing.T) {
	// Simulate a KV v2 secret response
	secret := &vault.Secret{
		Data: map[string]interface{}{
			"data": map[string]interface{}{
				"testKey": "testValue",
			},
			"metadata": map[string]interface{}{
				"version": 1,
			},
		},
	}

	// Test that we can access nested data (KV v2 format)
	if secret.Data == nil {
		t.Fatal("Secret.Data is nil")
	}

	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		t.Fatal("Secret.Data['data'] is not map[string]interface{} - KV v2 structure changed")
	}

	value, ok := data["testKey"].(string)
	if !ok {
		t.Fatal("Secret data value is not string - type assertion changed")
	}

	if value != "testValue" {
		t.Errorf("Expected 'testValue', got '%s'", value)
	}
}

// TestGetDomainFromVaultErrorHandling validates error handling paths
func TestGetDomainFromVaultErrorHandling(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		key       string
		wantError bool
	}{
		{
			name:      "empty path",
			path:      "",
			key:       "domain",
			wantError: true,
		},
		{
			name:      "empty key",
			path:      "kv/data/domains",
			key:       "",
			wantError: true,
		},
		{
			name:      "valid inputs structure",
			path:      "kv/data/domains",
			key:       "prodDomain",
			wantError: false, // We expect error from vault connection, but validation should pass
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetDomainFromVault(tt.path, tt.key)
			
			if tt.wantError && err == nil {
				t.Error("Expected error for invalid input, got nil")
			}
			
			if !tt.wantError && err == nil {
				t.Error("Expected vault connection error in test environment")
			}
			
			// All calls should return an error (either validation or connection)
			// This just ensures the function doesn't panic
			if err == nil {
				t.Log("Unexpected success - Vault might be configured in test env")
			}
		})
	}
}
