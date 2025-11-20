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

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	networkingv1 "github.com/floryn08/homelab-alm/api/v1"
	"github.com/floryn08/homelab-alm/internal/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CertificateRequestReconciler reconciles a CertificateRequest object
type CertificateRequestReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=networking.alm.homelab,resources=certificaterequests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.alm.homelab,resources=certificaterequests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.alm.homelab,resources=certificaterequests/finalizers,verbs=update
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete

func (r *CertificateRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the CertificateRequest CR
	var cr networkingv1.CertificateRequest
	if err := r.Get(ctx, req.NamespacedName, &cr); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil // CR deleted, nothing to do
		}
		logger.Error(err, "failed to get CertificateRequest")
		return ctrl.Result{}, err
	}

	// Fetch domain from Vault
	fqdn, err := r.getFQDN(&cr)
	if err != nil {
		logger.Error(err, "failed to construct FQDN")
		return ctrl.Result{}, err
	}

	// Create or update the Certificate
	cert := r.buildCertificate(&cr, fqdn)
	if err := ctrl.SetControllerReference(&cr, cert, r.Scheme); err != nil {
		logger.Error(err, "failed to set controller reference")
		return ctrl.Result{}, err
	}

	if err := r.createOrUpdateCertificate(ctx, cert); err != nil {
		logger.Error(err, "failed to create or update Certificate")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled Certificate", "fqdn", fqdn)

	// Update status
	return r.updateStatus(ctx, &cr, fqdn)
}

// getFQDN constructs the FQDN by fetching the domain from Vault
func (r *CertificateRequestReconciler) getFQDN(cr *networkingv1.CertificateRequest) (string, error) {
	vaultPath := cr.Spec.VaultPath
	if vaultPath == "" {
		vaultPath = "kv/data/domains"
	}

	domain, err := utils.GetDomainFromVault(vaultPath, cr.Spec.DomainKey)
	if err != nil {
		return "", fmt.Errorf("failed to get domain from Vault: %w", err)
	}

	if cr.Spec.Subdomain == "" {
		return domain, nil
	}

	return fmt.Sprintf("%s.%s", cr.Spec.Subdomain, domain), nil
}

// buildCertificate constructs the desired Certificate resource
func (r *CertificateRequestReconciler) buildCertificate(cr *networkingv1.CertificateRequest, fqdn string) *certmanagerv1.Certificate {
	issuerName := cr.Spec.IssuerName
	if issuerName == "" {
		issuerName = "ca-issuer"
	}

	issuerKind := cr.Spec.IssuerKind
	if issuerKind == "" {
		issuerKind = "ClusterIssuer"
	}

	// Extract base domain for organization
	domain := fqdn
	if cr.Spec.Subdomain != "" {
		// Remove subdomain to get base domain
		domain = fqdn[len(cr.Spec.Subdomain)+1:]
	}

	return &certmanagerv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-certificate",
			Namespace: cr.Namespace,
		},
		Spec: certmanagerv1.CertificateSpec{
			SecretName:           cr.Spec.SecretName,
			RevisionHistoryLimit: int32Ptr(1),
			IssuerRef: cmmeta.ObjectReference{
				Name: issuerName,
				Kind: issuerKind,
			},
			CommonName: fqdn,
			DNSNames:   []string{fqdn},
			Subject: &certmanagerv1.X509Subject{
				Organizations:       []string{domain},
				OrganizationalUnits: []string{cr.Namespace},
			},
		},
	}
}

// createOrUpdateCertificate creates or updates the Certificate resource
func (r *CertificateRequestReconciler) createOrUpdateCertificate(ctx context.Context, cert *certmanagerv1.Certificate) error {
	logger := log.FromContext(ctx)

	var existing certmanagerv1.Certificate
	err := r.Get(ctx, client.ObjectKeyFromObject(cert), &existing)

	if errors.IsNotFound(err) {
		if err := r.Create(ctx, cert); err != nil {
			return fmt.Errorf("failed to create Certificate: %w", err)
		}
		logger.Info("Created Certificate", "name", cert.Name, "namespace", cert.Namespace)
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to get existing Certificate: %w", err)
	}

	// Update existing certificate
	cert.ResourceVersion = existing.ResourceVersion
	if err := r.Update(ctx, cert); err != nil {
		return fmt.Errorf("failed to update Certificate: %w", err)
	}

	logger.Info("Updated Certificate", "name", cert.Name, "namespace", cert.Namespace)
	return nil
}

// updateStatus updates the CertificateRequest status
func (r *CertificateRequestReconciler) updateStatus(ctx context.Context, cr *networkingv1.CertificateRequest, fqdn string) (ctrl.Result, error) {
	cr.Status.FQDN = fqdn
	cr.Status.Ready = true

	if err := r.Status().Update(ctx, cr); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update status: %w", err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CertificateRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1.CertificateRequest{}).
		Named("certificaterequest").
		Complete(r)
}

func int32Ptr(i int32) *int32 {
	return &i
}
