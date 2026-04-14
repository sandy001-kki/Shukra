// This file builds the Ingress resource for an AppEnvironment. It exists so
// ingress-specific path, class, and TLS handling are kept out of the controller.
package resources

import (
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
)

func Ingress(appEnv *appsv1beta1.AppEnvironment) *networkingv1.Ingress {
	pathType := networkingv1.PathTypePrefix
	if appEnv.Spec.Ingress.PathType != nil {
		pathType = *appEnv.Spec.Ingress.PathType
	}
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        Name(appEnv, "ingress"),
			Namespace:   appEnv.Namespace,
			Labels:      appEnv.Labels("ingress"),
			Annotations: appEnv.Spec.Ingress.Annotations,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: stringPtr(appEnv.Spec.Ingress.ClassName),
			Rules: []networkingv1.IngressRule{{
				Host: appEnv.Spec.Ingress.Host,
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							Path:     appEnv.Spec.Ingress.Path,
							PathType: &pathType,
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: Name(appEnv, "service"),
									Port: networkingv1.ServiceBackendPort{Number: appEnv.ServicePort()},
								},
							},
						}},
					},
				},
			}},
		},
	}
	if appEnv.Spec.Ingress.TLSSecretName != "" {
		ing.Spec.TLS = []networkingv1.IngressTLS{{
			SecretName: appEnv.Spec.Ingress.TLSSecretName,
			Hosts:      []string{appEnv.Spec.Ingress.Host},
		}}
	}
	return ing
}

func stringPtr(value string) *string {
	return &value
}
