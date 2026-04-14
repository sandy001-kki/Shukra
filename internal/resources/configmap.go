// This file builds the desired ConfigMap for application configuration. It
// exists as a builder-only package so reconciliation code stays orchestration-focused.
package resources

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
)

func ConfigMap(appEnv *appsv1beta1.AppEnvironment) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        Name(appEnv, "cm"),
			Namespace:   appEnv.Namespace,
			Labels:      appEnv.Labels("configmap"),
			Annotations: map[string]string{},
		},
		Data: appEnv.Spec.Config.Data,
	}
}
