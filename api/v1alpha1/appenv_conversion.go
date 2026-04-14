// This file implements spoke conversion from v1alpha1 to the v1beta1 hub and
// back. It exists because the two versions have structural differences for
// secret references and network policy configuration.
package v1alpha1

import (
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
	conversion "sigs.k8s.io/controller-runtime/pkg/conversion"

	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
)

func (src *AppEnvironment) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*appsv1beta1.AppEnvironment)
	if !ok {
		return fmt.Errorf("expected v1beta1.AppEnvironment hub")
	}

	dst.ObjectMeta = src.ObjectMeta
	dst.Spec.App = appsv1beta1.AppSpec{
		Image:              src.Spec.App.Image,
		ImagePullPolicy:    src.Spec.App.ImagePullPolicy,
		Replicas:           src.Spec.App.Replicas,
		ContainerPort:      src.Spec.App.ContainerPort,
		Env:                src.Spec.App.Env,
		EnvFrom:            src.Spec.App.EnvFrom,
		Resources:          src.Spec.App.Resources,
		Strategy:           src.Spec.App.Strategy,
		LivenessProbe:      src.Spec.App.LivenessProbe,
		ReadinessProbe:     src.Spec.App.ReadinessProbe,
		StartupProbe:       src.Spec.App.StartupProbe,
		ServiceAccountName: src.Spec.App.ServiceAccountName,
	}
	for _, secretName := range src.Spec.App.SecretRefs {
		dst.Spec.App.SecretRefs = append(dst.Spec.App.SecretRefs, appsv1beta1.SecretRef{
			Name:    secretName,
			MountAs: "env",
		})
	}
	dst.Spec.Config = appsv1beta1.ConfigSpec{Data: src.Spec.Config.Data}
	dst.Spec.Service = appsv1beta1.ServiceSpec(src.Spec.Service)
	pathType := networkingv1.PathTypePrefix
	if src.Spec.Ingress.PathType == "Exact" {
		pathType = networkingv1.PathTypeExact
	}
	dst.Spec.Ingress = appsv1beta1.IngressSpec{
		Enabled:                src.Spec.Ingress.Enabled,
		Host:                   src.Spec.Ingress.Host,
		Path:                   src.Spec.Ingress.Path,
		PathType:               &pathType,
		ClassName:              src.Spec.Ingress.ClassName,
		TLSSecretName:          src.Spec.Ingress.TLSSecretName,
		Annotations:            src.Spec.Ingress.Annotations,
		AllowSharedIngressHost: src.Spec.Ingress.AllowSharedIngressHost,
	}
	dst.Spec.Database = appsv1beta1.DatabaseSpec(src.Spec.Database)
	dst.Spec.Migration = appsv1beta1.MigrationSpec(src.Spec.Migration)
	dst.Spec.Autoscaling = appsv1beta1.AutoscalingSpec(src.Spec.Autoscaling)
	dst.Spec.Backup = appsv1beta1.BackupSpec(src.Spec.Backup)
	dst.Spec.Restore = appsv1beta1.RestoreSpec(src.Spec.Restore)
	if src.Spec.Security.NetworkPolicy {
		dst.Spec.Security.NetworkPolicy = appsv1beta1.NetworkPolicySpec{
			IngressRules: []networkingv1.NetworkPolicyIngressRule{{}},
		}
	}
	dst.Spec.Security.PodDisruptionBudget = appsv1beta1.PDBSpec{
		Enabled: src.Spec.Security.PodDisruptionBudget.Enabled,
	}
	dst.Spec.Paused = src.Spec.Paused
	dst.Status = appsv1beta1.AppEnvironmentStatus{
		Phase:                       src.Status.Phase,
		ObservedGeneration:          src.Status.ObservedGeneration,
		URL:                         src.Status.URL,
		ChildResources:              appsv1beta1.ChildResources(src.Status.ChildResources),
		LastError:                   src.Status.LastError,
		FailureCount:                src.Status.FailureCount,
		LastAppliedMigrationID:      src.Status.LastAppliedMigrationID,
		LastProcessedRestoreNonce:   src.Status.LastProcessedRestoreNonce,
		LastSuccessfulReconcileTime: src.Status.LastSuccessfulReconcileTime,
		LastAppliedSpecHash:         src.Status.LastAppliedSpecHash,
		Conditions:                  src.Status.Conditions,
	}
	return nil
}

func (dst *AppEnvironment) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*appsv1beta1.AppEnvironment)
	if !ok {
		return fmt.Errorf("expected v1beta1.AppEnvironment hub")
	}

	dst.ObjectMeta = src.ObjectMeta
	dst.Spec.App = AppSpec{
		Image:              src.Spec.App.Image,
		ImagePullPolicy:    src.Spec.App.ImagePullPolicy,
		Replicas:           src.Spec.App.Replicas,
		ContainerPort:      src.Spec.App.ContainerPort,
		Env:                src.Spec.App.Env,
		EnvFrom:            src.Spec.App.EnvFrom,
		Resources:          src.Spec.App.Resources,
		Strategy:           src.Spec.App.Strategy,
		LivenessProbe:      src.Spec.App.LivenessProbe,
		ReadinessProbe:     src.Spec.App.ReadinessProbe,
		StartupProbe:       src.Spec.App.StartupProbe,
		ServiceAccountName: src.Spec.App.ServiceAccountName,
	}
	for _, secretRef := range src.Spec.App.SecretRefs {
		dst.Spec.App.SecretRefs = append(dst.Spec.App.SecretRefs, secretRef.Name)
	}
	pathType := ""
	if src.Spec.Ingress.PathType != nil {
		pathType = string(*src.Spec.Ingress.PathType)
	}
	dst.Spec.Config = ConfigSpec{Data: src.Spec.Config.Data}
	dst.Spec.Service = ServiceSpec(src.Spec.Service)
	dst.Spec.Ingress = IngressSpec{
		Enabled:                src.Spec.Ingress.Enabled,
		Host:                   src.Spec.Ingress.Host,
		Path:                   src.Spec.Ingress.Path,
		PathType:               pathType,
		ClassName:              src.Spec.Ingress.ClassName,
		TLSSecretName:          src.Spec.Ingress.TLSSecretName,
		Annotations:            src.Spec.Ingress.Annotations,
		AllowSharedIngressHost: src.Spec.Ingress.AllowSharedIngressHost,
	}
	dst.Spec.Database = DatabaseSpec(src.Spec.Database)
	dst.Spec.Migration = MigrationSpec(src.Spec.Migration)
	dst.Spec.Autoscaling = AutoscalingSpec(src.Spec.Autoscaling)
	dst.Spec.Backup = BackupSpec(src.Spec.Backup)
	dst.Spec.Restore = RestoreSpec(src.Spec.Restore)
	dst.Spec.Security.NetworkPolicy = len(src.Spec.Security.NetworkPolicy.IngressRules) > 0
	dst.Spec.Security.PodDisruptionBudget.Enabled = src.Spec.Security.PodDisruptionBudget.Enabled
	dst.Spec.Paused = src.Spec.Paused
	if dst.Annotations == nil {
		dst.Annotations = map[string]string{}
	}
	dst.Annotations["conversion.shukra.io/downgrade-lossy"] = "true"
	dst.Status = AppEnvironmentStatus{
		Phase:                       src.Status.Phase,
		ObservedGeneration:          src.Status.ObservedGeneration,
		URL:                         src.Status.URL,
		ChildResources:              ChildResources(src.Status.ChildResources),
		LastError:                   src.Status.LastError,
		FailureCount:                src.Status.FailureCount,
		LastAppliedMigrationID:      src.Status.LastAppliedMigrationID,
		LastProcessedRestoreNonce:   src.Status.LastProcessedRestoreNonce,
		LastSuccessfulReconcileTime: src.Status.LastSuccessfulReconcileTime,
		LastAppliedSpecHash:         src.Status.LastAppliedSpecHash,
		Conditions:                  src.Status.Conditions,
	}
	return nil
}
