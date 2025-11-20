package utils

import (
	"fmt"
	"sync"

	vault "github.com/hashicorp/vault/api"
)

var (
	vaultClient     *vault.Client
	vaultClientOnce sync.Once
	vaultClientErr  error
)

// GetVaultClient returns a singleton Vault client instance
func GetVaultClient() (*vault.Client, error) {
	vaultClientOnce.Do(func() {
		config := vault.DefaultConfig()
		if config.Error != nil {
			vaultClientErr = fmt.Errorf("failed to create Vault config: %w", config.Error)
			return
		}

		client, err := vault.NewClient(config)
		if err != nil {
			vaultClientErr = fmt.Errorf("failed to create Vault client: %w", err)
			return
		}

		vaultClient = client
	})

	return vaultClient, vaultClientErr
}

// GetDomainFromVault fetches a domain value from Vault at the specified path and key
func GetDomainFromVault(path, key string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("vault path cannot be empty")
	}
	if key == "" {
		return "", fmt.Errorf("domain key cannot be empty")
	}

	client, err := GetVaultClient()
	if err != nil {
		return "", fmt.Errorf("failed to get Vault client: %w", err)
	}

	secret, err := client.Logical().Read(path)
	if err != nil {
		return "", fmt.Errorf("failed to read secret from Vault at path %s: %w", path, err)
	}

	if secret == nil {
		return "", fmt.Errorf("no secret found at Vault path %s", path)
	}

	if secret.Data == nil {
		return "", fmt.Errorf("secret at path %s has no data", path)
	}

	// For KV v2, data is nested under "data" key
	dataRaw, ok := secret.Data["data"]
	if !ok {
		return "", fmt.Errorf("secret at path %s is missing 'data' field (ensure you're using KV v2 path format)", path)
	}

	data, ok := dataRaw.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("secret data at path %s is not a map", path)
	}

	valRaw, ok := data[key]
	if !ok {
		return "", fmt.Errorf("domain key '%s' not found in secret at path %s", key, path)
	}

	val, ok := valRaw.(string)
	if !ok {
		return "", fmt.Errorf("domain key '%s' at path %s is not a string (got type %T)", key, path, valRaw)
	}

	if val == "" {
		return "", fmt.Errorf("domain key '%s' at path %s is empty", key, path)
	}

	return val, nil
}
