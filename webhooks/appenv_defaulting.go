// This file implements mutating webhook defaulting for AppEnvironment. It
// exists so users can submit minimal specs while the operator enforces sane
// production-friendly defaults consistently before reconciliation.
package webhooks

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
)

type AppEnvironmentDefaulter struct{}

func (d *AppEnvironmentDefaulter) Default(_ context.Context, obj runtime.Object) error {
	appEnv := obj.(*appsv1beta1.AppEnvironment)
	if appEnv.Spec.App.Replicas == nil {
		appEnv.Spec.App.Replicas = ptr(int32(2))
	}
	if appEnv.Spec.App.ImagePullPolicy == "" {
		appEnv.Spec.App.ImagePullPolicy = corev1.PullIfNotPresent
	}
	if appEnv.Spec.App.ContainerPort == 0 {
		appEnv.Spec.App.ContainerPort = 8080
	}
	if appEnv.Spec.Service.Enabled == nil {
		appEnv.Spec.Service.Enabled = ptr(true)
	}
	if appEnv.Spec.Service.Type == "" {
		appEnv.Spec.Service.Type = corev1.ServiceTypeClusterIP
	}
	if appEnv.Spec.Service.Port == 0 {
		appEnv.Spec.Service.Port = 80
	}
	if appEnv.Spec.Ingress.Path == "" {
		appEnv.Spec.Ingress.Path = "/"
	}
	if appEnv.Spec.Ingress.PathType == nil {
		pathType := networkingv1.PathTypePrefix
		appEnv.Spec.Ingress.PathType = &pathType
	}
	if !appEnv.Spec.Ingress.AllowSharedIngressHost {
		appEnv.Spec.Ingress.AllowSharedIngressHost = false
	}
	if appEnv.Spec.Migration.BackoffLimit == nil {
		appEnv.Spec.Migration.BackoffLimit = ptr(int32(3))
	}
	if appEnv.Spec.Migration.ActiveDeadlineSeconds == nil {
		appEnv.Spec.Migration.ActiveDeadlineSeconds = ptr(int64(300))
	}
	if appEnv.Spec.App.LivenessProbe == nil {
		appEnv.Spec.App.LivenessProbe = httpProbe(appEnv.Spec.App.ContainerPort, 10)
	}
	if appEnv.Spec.App.ReadinessProbe == nil {
		appEnv.Spec.App.ReadinessProbe = httpProbe(appEnv.Spec.App.ContainerPort, 5)
	}
	return nil
}

func httpProbe(port int32, delay int32) *corev1.Probe {
	return &corev1.Probe{
		InitialDelaySeconds: delay,
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/",
				Port: intstrFromInt32(port),
			},
		},
	}
}

func intstrFromInt32(port int32) intstr.IntOrString {
	return intstr.FromInt32(port)
}

func ptr[T any](value T) *T { return &value }

var _ webhook.CustomDefaulter = &AppEnvironmentDefaulter{}

func setupDefaulter(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&appsv1beta1.AppEnvironment{}).WithDefaulter(&AppEnvironmentDefaulter{}).Complete()
}
