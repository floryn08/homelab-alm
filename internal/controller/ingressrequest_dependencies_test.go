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

	networkingv1 "github.com/floryn08/homelab-alm/api/v1"
	traefikv1alpha1 "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestBuildIngressRoute validates Traefik IngressRoute creation logic
func TestBuildIngressRoute(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = networkingv1.AddToScheme(scheme)
	_ = traefikv1alpha1.AddToScheme(scheme)

	reconciler := &IngressRequestReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
		Scheme: scheme,
	}

	tests := []struct {
		name            string
		ir              *networkingv1.IngressRequest
		fqdn            string
		wantEntrypoints []string
		wantServiceName string
		wantServicePort string
	}{
		{
			name: "default entrypoint",
			ir: &networkingv1.IngressRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ingress",
					Namespace: "default",
				},
				Spec: networkingv1.IngressRequestSpec{
					Subdomain:   "app",
					ServiceName: "app-service",
					ServicePort: "http",
					DomainKey:   "prodDomain",
				},
			},
			fqdn:            "app.example.com",
			wantEntrypoints: []string{"web"},
			wantServiceName: "app-service",
			wantServicePort: "http",
		},
		{
			name: "custom entrypoints",
			ir: &networkingv1.IngressRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ingress",
					Namespace: "default",
				},
				Spec: networkingv1.IngressRequestSpec{
					Subdomain:   "app",
					ServiceName: "app-service",
					ServicePort: "8080",
					DomainKey:   "prodDomain",
					Entrypoints: []string{"websecure"},
				},
			},
			fqdn:            "app.example.com",
			wantEntrypoints: []string{"websecure"},
			wantServiceName: "app-service",
			wantServicePort: "8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := reconciler.buildIngressRoute(tt.ir, tt.fqdn)

			// Validate Traefik API structure
			if route == nil {
				t.Fatal("buildIngressRoute returned nil")
			}

			if len(route.Spec.EntryPoints) != len(tt.wantEntrypoints) {
				t.Errorf("EntryPoints count = %v, want %v", len(route.Spec.EntryPoints), len(tt.wantEntrypoints))
			}

			if len(route.Spec.Routes) != 1 {
				t.Errorf("Routes count = %v, want 1", len(route.Spec.Routes))
			}

			routeSpec := route.Spec.Routes[0]
			expectedMatch := "Host(`" + tt.fqdn + "`)"
			if routeSpec.Match != expectedMatch {
				t.Errorf("Route.Match = %v, want %v", routeSpec.Match, expectedMatch)
			}

			if routeSpec.Kind != "Rule" {
				t.Errorf("Route.Kind = %v, want Rule", routeSpec.Kind)
			}

			if len(routeSpec.Services) != 1 {
				t.Errorf("Services count = %v, want 1", len(routeSpec.Services))
			}

			service := routeSpec.Services[0]
			if service.LoadBalancerSpec.Name != tt.wantServiceName {
				t.Errorf("Service.Name = %v, want %v", service.LoadBalancerSpec.Name, tt.wantServiceName)
			}

			portStr := service.LoadBalancerSpec.Port.StrVal
			if portStr != tt.wantServicePort {
				t.Errorf("Service.Port = %v, want %v", portStr, tt.wantServicePort)
			}
		})
	}
}

// TestBuildServices validates service configuration
func TestBuildServices(t *testing.T) {
	reconciler := &IngressRequestReconciler{}

	ir := &networkingv1.IngressRequest{
		Spec: networkingv1.IngressRequestSpec{
			ServiceName: "test-svc",
			ServicePort: "http",
		},
	}

	services := reconciler.buildServices(ir)

	if len(services) != 1 {
		t.Errorf("buildServices returned %d services, want 1", len(services))
	}

	svc := services[0]
	if svc.LoadBalancerSpec.Name != "test-svc" {
		t.Errorf("Service.Name = %v, want test-svc", svc.LoadBalancerSpec.Name)
	}

	// Validate Port is intstr type (Traefik API requirement)
	port := svc.LoadBalancerSpec.Port
	if port.Type != intstr.String {
		t.Errorf("Port.Type = %v, want String", port.Type)
	}
	if port.StrVal != "http" {
		t.Errorf("Port.StrVal = %v, want http", port.StrVal)
	}
}

// TestBuildMiddlewares validates middleware conversion
func TestBuildMiddlewares(t *testing.T) {
	reconciler := &IngressRequestReconciler{}

	tests := []struct {
		name  string
		ir    *networkingv1.IngressRequest
		count int
	}{
		{
			name: "no middlewares",
			ir: &networkingv1.IngressRequest{
				Spec: networkingv1.IngressRequestSpec{},
			},
			count: 0,
		},
		{
			name: "with middlewares",
			ir: &networkingv1.IngressRequest{
				Spec: networkingv1.IngressRequestSpec{
					Middlewares: []networkingv1.MiddlewareRef{
						{Name: "auth", Namespace: "default"},
						{Name: "rate-limit", Namespace: "middleware-ns"},
					},
				},
			},
			count: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middlewares := reconciler.buildMiddlewares(tt.ir)

			if len(middlewares) != tt.count {
				t.Errorf("buildMiddlewares returned %d middlewares, want %d", len(middlewares), tt.count)
			}

			// Validate Traefik MiddlewareRef structure
			if tt.count > 0 {
				mw := middlewares[0]
				if mw.Name == "" {
					t.Error("MiddlewareRef.Name is empty - Traefik API may have changed")
				}
				if mw.Namespace == "" {
					t.Error("MiddlewareRef.Namespace is empty - Traefik API may have changed")
				}
			}
		})
	}
}

// TestBuildTLSConfig validates TLS configuration
func TestBuildTLSConfig(t *testing.T) {
	reconciler := &IngressRequestReconciler{}

	tests := []struct {
		name             string
		tlsSpec          *networkingv1.IngressTLSConfig
		wantSecretName   string
		wantCertResolver string
	}{
		{
			name: "with secret",
			tlsSpec: &networkingv1.IngressTLSConfig{
				SecretName: "my-tls-secret",
			},
			wantSecretName:   "my-tls-secret",
			wantCertResolver: "",
		},
		{
			name: "with cert resolver",
			tlsSpec: &networkingv1.IngressTLSConfig{
				CertResolver: "letsencrypt",
			},
			wantSecretName:   "",
			wantCertResolver: "letsencrypt",
		},
		{
			name: "with both",
			tlsSpec: &networkingv1.IngressTLSConfig{
				SecretName:   "my-tls-secret",
				CertResolver: "letsencrypt",
			},
			wantSecretName:   "my-tls-secret",
			wantCertResolver: "letsencrypt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tlsConfig := reconciler.buildTLSConfig(tt.tlsSpec)

			if tlsConfig == nil {
				t.Fatal("buildTLSConfig returned nil")
			}

			// Validate Traefik TLS structure
			if tlsConfig.SecretName != tt.wantSecretName {
				t.Errorf("TLS.SecretName = %v, want %v", tlsConfig.SecretName, tt.wantSecretName)
			}

			if tlsConfig.CertResolver != tt.wantCertResolver {
				t.Errorf("TLS.CertResolver = %v, want %v", tlsConfig.CertResolver, tt.wantCertResolver)
			}
		})
	}
}

// TestTraefikIngressRouteStructure validates core Traefik types
func TestTraefikIngressRouteStructure(t *testing.T) {
	// Test that we can create IngressRoute with expected structure
	route := &traefikv1alpha1.IngressRoute{
		Spec: traefikv1alpha1.IngressRouteSpec{
			EntryPoints: []string{"web"},
			Routes: []traefikv1alpha1.Route{
				{
					Match: "Host(`example.com`)",
					Kind:  "Rule",
					Services: []traefikv1alpha1.Service{
						{
							LoadBalancerSpec: traefikv1alpha1.LoadBalancerSpec{
								Name: "test-svc",
								Port: intstr.FromString("http"),
							},
						},
					},
				},
			},
		},
	}

	// Validate all fields are accessible
	if len(route.Spec.EntryPoints) == 0 {
		t.Error("IngressRoute.Spec.EntryPoints not accessible")
	}
	if len(route.Spec.Routes) == 0 {
		t.Error("IngressRoute.Spec.Routes not accessible")
	}
	if route.Spec.Routes[0].Match == "" {
		t.Error("Route.Match not accessible")
	}
	if route.Spec.Routes[0].Kind == "" {
		t.Error("Route.Kind not accessible")
	}
	if len(route.Spec.Routes[0].Services) == 0 {
		t.Error("Route.Services not accessible")
	}
}

// TestTraefikTLSStructure validates Traefik TLS configuration
func TestTraefikTLSStructure(t *testing.T) {
	tls := &traefikv1alpha1.TLS{
		SecretName:   "test-secret",
		CertResolver: "letsencrypt",
		Options: &traefikv1alpha1.TLSOptionRef{
			Name:      "default",
			Namespace: "default",
		},
	}

	if tls.SecretName == "" {
		t.Error("TLS.SecretName not accessible - Traefik API may have changed")
	}
	if tls.CertResolver == "" {
		t.Error("TLS.CertResolver not accessible - Traefik API may have changed")
	}
	if tls.Options == nil {
		t.Error("TLS.Options not accessible - Traefik API may have changed")
	}
}
