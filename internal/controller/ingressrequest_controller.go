package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	traefikv1alpha1 "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"

	networkingv1 "github.com/floryn08/homelab-alm/api/v1"
	"github.com/floryn08/homelab-alm/internal/utils"
)

// IngressRequestReconciler reconciles a IngressRequest object
type IngressRequestReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=networking.alm.homelab,resources=ingressrequests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.alm.homelab,resources=ingressrequests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.alm.homelab,resources=ingressrequests/finalizers,verbs=update
// +kubebuilder:rbac:groups=traefik.io,resources=ingressroutes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=traefik.io,resources=middlewares,verbs=get;list;watch

func (r *IngressRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the IngressRequest CR
	var ir networkingv1.IngressRequest
	if err := r.Get(ctx, req.NamespacedName, &ir); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil // CR deleted, nothing to do
		}
		logger.Error(err, "failed to get IngressRequest")
		return ctrl.Result{}, err
	}

	// Fetch domain from Vault and construct FQDN
	fqdn, err := r.getFQDN(&ir)
	if err != nil {
		logger.Error(err, "failed to construct FQDN")
		return ctrl.Result{}, err
	}

	// Build the IngressRoute
	route := r.buildIngressRoute(&ir, fqdn)
	if err := ctrl.SetControllerReference(&ir, route, r.Scheme); err != nil {
		logger.Error(err, "failed to set controller reference")
		return ctrl.Result{}, err
	}

	// Create or update the IngressRoute
	if err := r.createOrUpdateIngressRoute(ctx, route); err != nil {
		logger.Error(err, "failed to create or update IngressRoute")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled IngressRoute", "fqdn", fqdn)

	// Update status
	return r.updateStatus(ctx, &ir, fqdn)
}

// getFQDN constructs the FQDN by fetching the domain from Vault
func (r *IngressRequestReconciler) getFQDN(ir *networkingv1.IngressRequest) (string, error) {
	vaultPath := ir.Spec.VaultPath
	if vaultPath == "" {
		vaultPath = "kv/data/domains"
	}

	domain, err := utils.GetDomainFromVault(vaultPath, ir.Spec.DomainKey)
	if err != nil {
		return "", fmt.Errorf("failed to get domain from Vault: %w", err)
	}

	return fmt.Sprintf("%s.%s", ir.Spec.Subdomain, domain), nil
}

// buildIngressRoute constructs the desired IngressRoute resource
func (r *IngressRequestReconciler) buildIngressRoute(ir *networkingv1.IngressRequest, fqdn string) *traefikv1alpha1.IngressRoute {
	entrypoints := ir.Spec.Entrypoints
	if len(entrypoints) == 0 {
		entrypoints = []string{"web"}
	}

	route := &traefikv1alpha1.IngressRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ir.Name,
			Namespace: ir.Namespace,
		},
		Spec: traefikv1alpha1.IngressRouteSpec{
			EntryPoints: entrypoints,
			Routes: []traefikv1alpha1.Route{
				{
					Match:       fmt.Sprintf("Host(`%s`)", fqdn),
					Kind:        "Rule",
					Services:    r.buildServices(ir),
					Middlewares: r.buildMiddlewares(ir),
				},
			},
		},
	}

	if ir.Spec.TLS != nil {
		route.Spec.TLS = r.buildTLSConfig(ir.Spec.TLS)
	}

	return route
}

// buildServices creates the service configuration for the IngressRoute
func (r *IngressRequestReconciler) buildServices(ir *networkingv1.IngressRequest) []traefikv1alpha1.Service {
	return []traefikv1alpha1.Service{
		{
			LoadBalancerSpec: traefikv1alpha1.LoadBalancerSpec{
				Name: ir.Spec.ServiceName,
				Port: intstr.FromString(ir.Spec.ServicePort),
			},
		},
	}
}

// buildMiddlewares converts middleware references
func (r *IngressRequestReconciler) buildMiddlewares(ir *networkingv1.IngressRequest) []traefikv1alpha1.MiddlewareRef {
	middlewares := make([]traefikv1alpha1.MiddlewareRef, 0, len(ir.Spec.Middlewares))
	for _, mw := range ir.Spec.Middlewares {
		middlewares = append(middlewares, traefikv1alpha1.MiddlewareRef{
			Name:      mw.Name,
			Namespace: mw.Namespace,
		})
	}
	return middlewares
}

// buildTLSConfig creates TLS configuration for the IngressRoute
func (r *IngressRequestReconciler) buildTLSConfig(tlsSpec *networkingv1.IngressTLSConfig) *traefikv1alpha1.TLS {
	tlsConfig := &traefikv1alpha1.TLS{}

	if tlsSpec.SecretName != "" {
		tlsConfig.SecretName = tlsSpec.SecretName
	}

	if tlsSpec.CertResolver != "" {
		tlsConfig.CertResolver = tlsSpec.CertResolver
	}

	return tlsConfig
}

// createOrUpdateIngressRoute creates or updates the IngressRoute resource
func (r *IngressRequestReconciler) createOrUpdateIngressRoute(ctx context.Context, route *traefikv1alpha1.IngressRoute) error {
	logger := log.FromContext(ctx)

	var existing traefikv1alpha1.IngressRoute
	err := r.Get(ctx, client.ObjectKeyFromObject(route), &existing)

	if errors.IsNotFound(err) {
		if err := r.Create(ctx, route); err != nil {
			return fmt.Errorf("failed to create IngressRoute: %w", err)
		}
		logger.Info("Created IngressRoute", "name", route.Name, "namespace", route.Namespace)
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to get existing IngressRoute: %w", err)
	}

	// Update existing IngressRoute
	route.ResourceVersion = existing.ResourceVersion
	if err := r.Update(ctx, route); err != nil {
		return fmt.Errorf("failed to update IngressRoute: %w", err)
	}

	logger.Info("Updated IngressRoute", "name", route.Name, "namespace", route.Namespace)
	return nil
}

// updateStatus updates the IngressRequest status
func (r *IngressRequestReconciler) updateStatus(ctx context.Context, ir *networkingv1.IngressRequest, fqdn string) (ctrl.Result, error) {
	ir.Status.FQDN = fqdn

	if err := r.Status().Update(ctx, ir); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update status: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *IngressRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1.IngressRequest{}).
		Named("ingressrequest").
		Complete(r)
}
