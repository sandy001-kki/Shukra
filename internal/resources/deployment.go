// This file builds the application Deployment. It exists so Pod template logic,
// probes, resources, and secret mounting rules stay isolated from reconcile flow.
package resources

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
)

func Deployment(appEnv *appsv1beta1.AppEnvironment) *appsv1.Deployment {
	labels := appEnv.Labels("app")
	container := corev1.Container{
		Name:            "app",
		Image:           appEnv.Spec.App.Image,
		ImagePullPolicy: appEnv.Spec.App.ImagePullPolicy,
		Ports: []corev1.ContainerPort{{
			Name:          "http",
			ContainerPort: appEnv.EffectiveContainerPort(),
		}},
		Env:            appEnv.Spec.App.Env,
		EnvFrom:        buildEnvFrom(appEnv),
		Resources:      appEnv.Spec.App.Resources,
		LivenessProbe:  appEnv.Spec.App.LivenessProbe,
		ReadinessProbe: appEnv.Spec.App.ReadinessProbe,
		StartupProbe:   appEnv.Spec.App.StartupProbe,
		SecurityContext: appEnv.Spec.Security.ContainerSecurityContext,
		VolumeMounts:   buildVolumeMounts(appEnv),
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Name(appEnv, "deployment"),
			Namespace: appEnv.Namespace,
			Labels:    appEnv.Labels("deployment"),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(appEnv.EffectiveReplicas()),
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Strategy: appEnv.Spec.App.Strategy,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					ServiceAccountName: appEnv.Spec.App.ServiceAccountName,
					SecurityContext:    appEnv.Spec.Security.PodSecurityContext,
					Containers:         []corev1.Container{container},
					Volumes:            buildVolumes(appEnv),
				},
			},
		},
	}
}

func buildEnvFrom(appEnv *appsv1beta1.AppEnvironment) []corev1.EnvFromSource {
	envFrom := append([]corev1.EnvFromSource{}, appEnv.Spec.App.EnvFrom...)
	if len(appEnv.Spec.Config.Data) > 0 {
		envFrom = append(envFrom, corev1.EnvFromSource{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: Name(appEnv, "cm")},
			},
		})
	}
	for _, secretRef := range appEnv.Spec.App.SecretRefs {
		if secretRef.MountAs == "env" || secretRef.MountAs == "" {
			envFrom = append(envFrom, corev1.EnvFromSource{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: secretRef.Name},
				},
			})
		}
	}
	return envFrom
}

func buildVolumes(appEnv *appsv1beta1.AppEnvironment) []corev1.Volume {
	volumes := make([]corev1.Volume, 0)
	for _, secretRef := range appEnv.Spec.App.SecretRefs {
		if secretRef.MountAs == "volume" {
			volumes = append(volumes, corev1.Volume{
				Name: "secret-" + secretRef.Name,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{SecretName: secretRef.Name},
				},
			})
		}
	}
	return volumes
}

func buildVolumeMounts(appEnv *appsv1beta1.AppEnvironment) []corev1.VolumeMount {
	mounts := make([]corev1.VolumeMount, 0)
	for _, secretRef := range appEnv.Spec.App.SecretRefs {
		if secretRef.MountAs == "volume" {
			mounts = append(mounts, corev1.VolumeMount{
				Name:      "secret-" + secretRef.Name,
				MountPath: secretRef.MountPath,
				ReadOnly:  true,
			})
		}
	}
	return mounts
}

func int32Ptr(value int32) *int32 {
	return &value
}
