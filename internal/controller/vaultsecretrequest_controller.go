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

package controller

import (
	"context"
	"fmt"
	"time"

	networkingv1 "github.com/floryn08/homelab-alm/api/v1"
	"github.com/floryn08/homelab-alm/internal/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// VaultSecretRequestReconciler reconciles a VaultSecretRequest object
type VaultSecretRequestReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=networking.alm.homelab,resources=vaultsecretrequests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.alm.homelab,resources=vaultsecretrequests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.alm.homelab,resources=vaultsecretrequests/finalizers,verbs=update

func (r *VaultSecretRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the VaultSecretRequest CR
	var cr networkingv1.VaultSecretRequest
	if err := r.Get(ctx, req.NamespacedName, &cr); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "failed to get VaultSecretRequest")
		return ctrl.Result{}, err
	}

	mount := cr.Spec.Mount
	if mount == "" {
		mount = "kv"
	}
	vaultPath := fmt.Sprintf("%s/data/%s", mount, cr.Spec.Path)

	// Read existing secret from Vault
	existing, err := utils.ReadSecretFromVault(vaultPath)
	if err != nil {
		logger.Error(err, "failed to read existing secret from Vault", "path", vaultPath)
		return r.updateStatusError(ctx, &cr, fmt.Sprintf("failed to read from Vault: %v", err))
	}

	// If secret exists and overwrite is disabled, mark as synced
	if existing != nil && !cr.Spec.OverwriteExisting {
		logger.Info("Secret already exists in Vault, skipping (overwriteExisting=false)", "path", cr.Spec.Path)
		return r.updateStatusSynced(ctx, &cr, "secret already exists in Vault")
	}

	// Build the secret data
	secretData, err := r.buildSecretData(cr.Spec.SecretKeys, existing)
	if err != nil {
		logger.Error(err, "failed to build secret data")
		return r.updateStatusError(ctx, &cr, fmt.Sprintf("failed to generate secret values: %v", err))
	}

	// Write to Vault
	if err := utils.WriteSecretToVault(vaultPath, secretData); err != nil {
		logger.Error(err, "failed to write secret to Vault", "path", vaultPath)
		return r.updateStatusError(ctx, &cr, fmt.Sprintf("failed to write to Vault: %v", err))
	}

	logger.Info("Successfully synced secret to Vault", "path", cr.Spec.Path, "keys", len(cr.Spec.SecretKeys))
	return r.updateStatusSynced(ctx, &cr, "secret synced to Vault")
}

// buildSecretData constructs the final key-value map to write to Vault.
// For each key in the spec:
//   - If a value is provided, use it
//   - If value is empty and generateType is set:
//     - If the key already exists in Vault, keep the existing value
//     - Otherwise, generate a new value
//   - If value is empty and no generateType, use empty string
func (r *VaultSecretRequestReconciler) buildSecretData(
	secretKeys map[string]networkingv1.SecretKeyConfig,
	existing map[string]interface{},
) (map[string]interface{}, error) {
	result := make(map[string]interface{}, len(secretKeys))

	for key, config := range secretKeys {
		val, err := r.resolveKeyValue(key, config, existing)
		if err != nil {
			return nil, err
		}
		result[key] = val
	}

	return result, nil
}

// resolveKeyValue determines the final value for a single secret key.
func (r *VaultSecretRequestReconciler) resolveKeyValue(
	key string,
	config networkingv1.SecretKeyConfig,
	existing map[string]interface{},
) (string, error) {
	// Explicit value provided — always use it
	if config.Value != "" {
		return config.Value, nil
	}

	// No generate type — empty string
	if config.GenerateType == "" {
		return "", nil
	}

	// Check if key already exists in Vault — preserve it
	if existingVal := getExistingStringValue(existing, key); existingVal != "" {
		return existingVal, nil
	}

	// Generate a new value
	generated, err := utils.GenerateRandomValue(config.GenerateType, config.Length)
	if err != nil {
		return "", fmt.Errorf("failed to generate value for key %q: %w", key, err)
	}
	return generated, nil
}

func getExistingStringValue(existing map[string]interface{}, key string) string {
	if existing == nil {
		return ""
	}
	if val, ok := existing[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return ""
}

func (r *VaultSecretRequestReconciler) updateStatusSynced(ctx context.Context, cr *networkingv1.VaultSecretRequest, message string) (ctrl.Result, error) {
	now := metav1.NewTime(time.Now())
	cr.Status.Synced = true
	cr.Status.LastSyncedTime = &now
	cr.Status.Message = message

	if err := r.Status().Update(ctx, cr); err != nil {
		log.FromContext(ctx).Error(err, "failed to update VaultSecretRequest status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *VaultSecretRequestReconciler) updateStatusError(ctx context.Context, cr *networkingv1.VaultSecretRequest, message string) (ctrl.Result, error) {
	cr.Status.Synced = false
	cr.Status.Message = message

	if err := r.Status().Update(ctx, cr); err != nil {
		log.FromContext(ctx).Error(err, "failed to update VaultSecretRequest status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: 30 * time.Second}, fmt.Errorf("%s", message)
}

func (r *VaultSecretRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1.VaultSecretRequest{}).
		Named("vaultsecretrequest").
		Complete(r)
}
