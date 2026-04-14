// This file builds the Service managed for an AppEnvironment. It exists as a
// pure builder so the controller can reconcile Services with create-or-update.
package resources

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"

	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
)

func Service(appEnv *appsv1beta1.AppEnvironment) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        Name(appEnv, "service"),
			Namespace:   appEnv.Namespace,
			Labels:      appEnv.Labels("service"),
			Annotations: appEnv.Spec.Service.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Type:     appEnv.Spec.Service.Type,
			Selector: appEnv.Labels("app"),
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Port:       appEnv.ServicePort(),
				TargetPort: intstr.FromInt32(appEnv.ServiceTargetPort()),
			}},
		},
	}
}
