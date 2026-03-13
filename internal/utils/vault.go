package utils

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"

	vault "github.com/hashicorp/vault/api"
)

const (
	passwordCharset    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	errVaultPathEmpty   = "vault path cannot be empty"
	errVaultClientFail  = "failed to get Vault client: %w"
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
		return "", fmt.Errorf(errVaultPathEmpty)
	}
	if key == "" {
		return "", fmt.Errorf("domain key cannot be empty")
	}

	client, err := GetVaultClient()
	if err != nil {
		return "", fmt.Errorf(errVaultClientFail, err)
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

// ReadSecretFromVault reads a full secret from Vault KV v2 at the given path.
// Returns nil map if the secret does not exist.
func ReadSecretFromVault(path string) (map[string]interface{}, error) {
	if path == "" {
		return nil, fmt.Errorf(errVaultPathEmpty)
	}

	client, err := GetVaultClient()
	if err != nil {
		return nil, fmt.Errorf(errVaultClientFail, err)
	}

	secret, err := client.Logical().Read(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret from Vault at path %s: %w", path, err)
	}

	if secret == nil || secret.Data == nil {
		return nil, nil
	}

	dataRaw, ok := secret.Data["data"]
	if !ok {
		return nil, nil
	}

	data, ok := dataRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("secret data at path %s is not a map", path)
	}

	return data, nil
}

// WriteSecretToVault writes key-value pairs to Vault KV v2 at the given path.
func WriteSecretToVault(path string, data map[string]interface{}) error {
	if path == "" {
		return fmt.Errorf(errVaultPathEmpty)
	}

	client, err := GetVaultClient()
	if err != nil {
		return fmt.Errorf(errVaultClientFail, err)
	}

	wrappedData := map[string]interface{}{
		"data": data,
	}

	_, err = client.Logical().Write(path, wrappedData)
	if err != nil {
		return fmt.Errorf("failed to write secret to Vault at path %s: %w", path, err)
	}

	return nil
}

// GenerateRandomValue generates a random value based on the specified type and length.
func GenerateRandomValue(generateType string, length int) (string, error) {
	if length <= 0 {
		length = 32
	}

	switch generateType {
	case "password":
		return generatePassword(length)
	case "hex":
		return generateHex(length)
	case "uuid":
		return generateUUID()
	case "base64":
		return generateBase64(length)
	default:
		return "", fmt.Errorf("unsupported generate type: %s", generateType)
	}
}

func generatePassword(length int) (string, error) {
	result := make([]byte, length)
	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(passwordCharset))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random password: %w", err)
		}
		result[i] = passwordCharset[num.Int64()]
	}
	return string(result), nil
}

func generateHex(numBytes int) (string, error) {
	bytes := make([]byte, numBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random hex: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func generateUUID() (string, error) {
	uuid := make([]byte, 16)
	if _, err := rand.Read(uuid); err != nil {
		return "", fmt.Errorf("failed to generate UUID: %w", err)
	}
	// Set version 4 and variant bits
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16]), nil
}

func generateBase64(numBytes int) (string, error) {
	bytes := make([]byte, numBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random base64: %w", err)
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}
