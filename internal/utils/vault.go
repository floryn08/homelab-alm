package utils

import (
	"fmt"

	vault "github.com/hashicorp/vault/api"
)

func GetDomainFromVault(path, key string) (string, error) {
	config := vault.DefaultConfig()
	client, err := vault.NewClient(config)
	if err != nil {
		return "", err
	}

	secret, err := client.Logical().Read(path)
	if err != nil || secret == nil || secret.Data["data"] == nil {
		return "", fmt.Errorf("failed to read secret from Vault at path %s", path)
	}

	data := secret.Data["data"].(map[string]interface{})
	val, ok := data[key].(string)
	if !ok {
		return "", fmt.Errorf("domain key '%s' not found or invalid", key)
	}

	return val, nil
}
