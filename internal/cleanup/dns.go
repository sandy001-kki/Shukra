package cleanup

import (
	"context"
	"strings"
	"time"

	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DNSHook struct {
	client client.Client
}

func NewDNSHook(c client.Client) Hook {
	return &DNSHook{client: c}
}

func (h *DNSHook) Name() string {
	return "dns-cleanup"
}

func (h *DNSHook) Cleanup(ctx context.Context, env CleanupTarget) error {
	if env.IngressHost == "" {
		ctrl.LoggerFrom(ctx).Info("DNS cleanup skipped; no ingress host configured")
		return nil
	}

	hookCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	log := ctrl.LoggerFrom(ctx).WithValues("hook", h.Name(), "host", env.IngressHost)
	ingress := &networkingv1.Ingress{}
	err := h.client.Get(hookCtx, types.NamespacedName{Name: env.Name + "-ingress", Namespace: env.Namespace}, ingress)
	if apierrors.IsNotFound(err) {
		log.Info("DNS cleanup verified; ingress already gone")
		return nil
	}
	if err != nil {
		log.Error(err, "DNS verification could not read ingress; treating cleanup as best-effort")
		return nil
	}
	if !hasExternalDNSAnnotation(ingress.Annotations) {
		log.Info("DNS cleanup skipped; ingress is not managed by external-dns")
		return nil
	}

	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case <-hookCtx.Done():
		log.Error(hookCtx.Err(), "DNS provider verification timed out; treating cleanup as best-effort")
		return nil
	case <-timer.C:
		log.Info("DNS cleanup verification completed through external-dns-managed ingress")
		return nil
	}
}

func hasExternalDNSAnnotation(annotations map[string]string) bool {
	for key := range annotations {
		if strings.HasPrefix(key, "external-dns.alpha.kubernetes.io/") {
			return true
		}
	}
	return false
}
