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

func (r *IngressRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 1. Get the IngressRequest CR
	var ir networkingv1.IngressRequest
	if err := r.Get(ctx, req.NamespacedName, &ir); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil // CR deleted
		}
		return ctrl.Result{}, err
	}

	vaultPath := ir.Spec.VaultPath
	if vaultPath == "" {
		vaultPath = "kv/data/domains" // default
	}

	// 2. Fetch the domain from Vault using the provided domainKey
	domain, err := utils.GetDomainFromVault(vaultPath, ir.Spec.DomainKey)
	if err != nil {
		logger.Error(err, "failed to get domain from Vault")
		return ctrl.Result{}, err
	}

	// 3. Construct the full domain
	fqdn := fmt.Sprintf("%s.%s", ir.Spec.Subdomain, domain)

	entrypoints := []string{"web"}
	if len(ir.Spec.Entrypoints) > 0 {
		entrypoints = ir.Spec.Entrypoints
	}

	servicePort := intstr.FromString(ir.Spec.ServicePort)

	// 4. Define the desired IngressRoute
	route := &traefikv1alpha1.IngressRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ir.Name,
			Namespace: ir.Namespace,
		},
		Spec: traefikv1alpha1.IngressRouteSpec{
			EntryPoints: entrypoints,
			Routes: []traefikv1alpha1.Route{
				{
					Match: fmt.Sprintf("Host(`%s`)", fqdn),
					Kind:  "Rule",
					Services: []traefikv1alpha1.Service{
						{
							LoadBalancerSpec: traefikv1alpha1.LoadBalancerSpec{
								Name: ir.Spec.ServiceName,
								Port: servicePort,
							},
						},
					},
				},
			},
		},
	}

	if ir.Spec.TLS != nil {
		route.Spec.TLS = &traefikv1alpha1.TLS{
			SecretName: ir.Spec.TLS.SecretName,
		}
	}

	// 5. Set owner reference for cleanup
	if err := ctrl.SetControllerReference(&ir, route, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	// 6. Create or Update the IngressRoute
	var existing traefikv1alpha1.IngressRoute
	err = r.Get(ctx, req.NamespacedName, &existing)
	if errors.IsNotFound(err) {
		if err := r.Create(ctx, route); err != nil {
			logger.Error(err, "failed to create IngressRoute")
			return ctrl.Result{}, err
		}
		logger.Info("Created IngressRoute", "host", fqdn)
	} else if err == nil {
		route.ResourceVersion = existing.ResourceVersion
		if err := r.Update(ctx, route); err != nil {
			logger.Error(err, "failed to update IngressRoute")
			return ctrl.Result{}, err
		}
		logger.Info("Updated IngressRoute", "host", fqdn)
	} else {
		return ctrl.Result{}, err
	}

	// 7. Update status with FQDN
	ir.Status.FQDN = fqdn
	if err := r.Status().Update(ctx, &ir); err != nil {
		logger.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *IngressRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1.IngressRequest{}).
		Named("ingressrequest").
		Complete(r)
}
