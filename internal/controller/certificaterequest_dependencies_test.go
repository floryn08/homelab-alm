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
	"testing"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	networkingv1 "github.com/floryn08/homelab-alm/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestBuildCertificate validates cert-manager Certificate creation logic
func TestBuildCertificate(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = networkingv1.AddToScheme(scheme)
	_ = certmanagerv1.AddToScheme(scheme)

	reconciler := &CertificateRequestReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
		Scheme: scheme,
	}

	tests := []struct {
		name     string
		cr       *networkingv1.CertificateRequest
		fqdn     string
		wantName string
		wantKind string
	}{
		{
			name: "default issuer",
			cr: &networkingv1.CertificateRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cert",
					Namespace: "default",
				},
				Spec: networkingv1.CertificateRequestSpec{
					SecretName: "test-secret",
					DomainKey:  "prodDomain",
					Subdomain:  "app",
				},
			},
			fqdn:     "app.example.com",
			wantName: "ca-issuer",
			wantKind: "ClusterIssuer",
		},
		{
			name: "custom issuer",
			cr: &networkingv1.CertificateRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cert",
					Namespace: "default",
				},
				Spec: networkingv1.CertificateRequestSpec{
					SecretName: "test-secret",
					DomainKey:  "prodDomain",
					Subdomain:  "app",
					IssuerName: "letsencrypt",
					IssuerKind: "Issuer",
				},
			},
			fqdn:     "app.example.com",
			wantName: "letsencrypt",
			wantKind: "Issuer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cert := reconciler.buildCertificate(tt.cr, tt.fqdn)

			// Validate cert-manager API structure
			if cert == nil {
				t.Fatal("buildCertificate returned nil")
			}

			if cert.Spec.SecretName != tt.cr.Spec.SecretName {
				t.Errorf("SecretName = %v, want %v", cert.Spec.SecretName, tt.cr.Spec.SecretName)
			}

			if cert.Spec.IssuerRef.Name != tt.wantName {
				t.Errorf("IssuerRef.Name = %v, want %v", cert.Spec.IssuerRef.Name, tt.wantName)
			}

			if cert.Spec.IssuerRef.Kind != tt.wantKind {
				t.Errorf("IssuerRef.Kind = %v, want %v", cert.Spec.IssuerRef.Kind, tt.wantKind)
			}

			if cert.Spec.CommonName != tt.fqdn {
				t.Errorf("CommonName = %v, want %v", cert.Spec.CommonName, tt.fqdn)
			}

			if len(cert.Spec.DNSNames) != 1 || cert.Spec.DNSNames[0] != tt.fqdn {
				t.Errorf("DNSNames = %v, want [%v]", cert.Spec.DNSNames, tt.fqdn)
			}

			// Validate cert-manager specific fields we use
			if cert.Spec.Subject == nil {
				t.Error("Subject is nil - cert-manager API may have changed")
			}

			if cert.Spec.RevisionHistoryLimit == nil {
				t.Error("RevisionHistoryLimit is nil - cert-manager API may have changed")
			}
		})
	}
}

// TestCertManagerObjectReference validates the IssuerRef structure
func TestCertManagerObjectReference(t *testing.T) {
	// Test that we can create an ObjectReference with expected fields
	ref := cmmeta.ObjectReference{
		Name:  "test-issuer",
		Kind:  "ClusterIssuer",
		Group: "",
	}

	if ref.Name != "test-issuer" {
		t.Error("ObjectReference.Name field changed")
	}
	if ref.Kind != "ClusterIssuer" {
		t.Error("ObjectReference.Kind field changed")
	}

	// Test in Certificate context
	cert := &certmanagerv1.Certificate{
		Spec: certmanagerv1.CertificateSpec{
			IssuerRef: ref,
		},
	}

	if cert.Spec.IssuerRef.Name != "test-issuer" {
		t.Error("Certificate.Spec.IssuerRef.Name not accessible - API may have changed")
	}
}

// TestCertificateSubject validates X509Subject structure
func TestCertificateSubject(t *testing.T) {
	subject := &certmanagerv1.X509Subject{
		Organizations:       []string{"Example Org"},
		OrganizationalUnits: []string{"Engineering"},
		Countries:           []string{"US"},
		Provinces:           []string{"CA"},
		Localities:          []string{"San Francisco"},
	}

	if len(subject.Organizations) == 0 {
		t.Error("X509Subject.Organizations field missing or changed")
	}
	if len(subject.OrganizationalUnits) == 0 {
		t.Error("X509Subject.OrganizationalUnits field missing or changed")
	}

	// Test in Certificate context
	cert := &certmanagerv1.Certificate{
		Spec: certmanagerv1.CertificateSpec{
			Subject: subject,
		},
	}

	if cert.Spec.Subject == nil {
		t.Error("Certificate.Spec.Subject is nil - API may have changed")
	}
	if len(cert.Spec.Subject.Organizations) == 0 {
		t.Error("Certificate.Spec.Subject.Organizations not accessible")
	}
}

// TestInt32Ptr validates the helper function
func TestInt32Ptr(t *testing.T) {
	val := int32(1)
	ptr := int32Ptr(val)

	if ptr == nil {
		t.Error("int32Ptr returned nil")
	}
	if *ptr != val {
		t.Errorf("int32Ptr = %v, want %v", *ptr, val)
	}
}
