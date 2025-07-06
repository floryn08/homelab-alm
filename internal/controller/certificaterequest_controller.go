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

func (r *CertificateRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 1. Get the CertificateRequest CR
	var cr networkingv1.CertificateRequest
	if err := r.Get(ctx, req.NamespacedName, &cr); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil // CR deleted
		}
		return ctrl.Result{}, err
	}

	vaultPath := cr.Spec.VaultPath
	if vaultPath == "" {
		vaultPath = "kv/data/domains" // default
	}

	// 2. Get domain from Vault
	domain, err := utils.GetDomainFromVault(vaultPath, cr.Spec.DomainKey)
	if err != nil {
		logger.Error(err, "failed to get domain from Vault")
		return ctrl.Result{}, err
	}

	fqdn := fmt.Sprintf("%s.%s", cr.Spec.Subdomain, domain)

	// 3. Create desired Certificate
	cert := &certmanagerv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-certificate",
			Namespace: cr.Namespace,
		},
		Spec: certmanagerv1.CertificateSpec{
			SecretName:           cr.Spec.SecretName,
			RevisionHistoryLimit: int32Ptr(1),
			IssuerRef: cmmeta.ObjectReference{
				Name: "ca-issuer",
				Kind: "ClusterIssuer",
			},
			CommonName: fqdn,
			DNSNames:   []string{fqdn},
			Subject: &certmanagerv1.X509Subject{
				Organizations:       []string{domain},
				OrganizationalUnits: []string{cr.Namespace},
			},
		},
	}

	// Set owner reference so the cert is garbage collected when the CR is deleted
	if err := ctrl.SetControllerReference(&cr, cert, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	// 4. Create or update Certificate
	var existing certmanagerv1.Certificate
	err = r.Get(ctx, client.ObjectKeyFromObject(cert), &existing)
	if errors.IsNotFound(err) {
		if err := r.Create(ctx, cert); err != nil {
			logger.Error(err, "failed to create Certificate")
			return ctrl.Result{}, err
		}
		logger.Info("Created Certificate", "fqdn", fqdn)
	} else if err == nil {
		cert.ResourceVersion = existing.ResourceVersion
		if err := r.Update(ctx, cert); err != nil {
			logger.Error(err, "failed to update Certificate")
			return ctrl.Result{}, err
		}
		logger.Info("Updated Certificate", "fqdn", fqdn)
	} else {
		return ctrl.Result{}, err
	}

	// 5. Update status
	cr.Status.FQDN = fqdn
	cr.Status.Ready = true
	if err := r.Status().Update(ctx, &cr); err != nil {
		logger.Error(err, "failed to update status")
		return ctrl.Result{}, err
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
