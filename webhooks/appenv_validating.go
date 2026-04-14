// This file implements validating webhook rules for AppEnvironment. It exists
// to reject invalid or unsafe specs before they ever reach reconciliation.
package webhooks

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
)

var hostRegex = regexp.MustCompile(`^([a-z0-9]([-a-z0-9]*[a-z0-9])?\.)+[a-z]{2,}$`)

type AppEnvironmentValidator struct {
	Client client.Client
}

func (v *AppEnvironmentValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, v.validate(ctx, obj.(*appsv1beta1.AppEnvironment), nil)
}

func (v *AppEnvironmentValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	return nil, v.validate(ctx, newObj.(*appsv1beta1.AppEnvironment), oldObj.(*appsv1beta1.AppEnvironment))
}

func (v *AppEnvironmentValidator) ValidateDelete(context.Context, runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (v *AppEnvironmentValidator) validate(ctx context.Context, appEnv *appsv1beta1.AppEnvironment, old *appsv1beta1.AppEnvironment) error {
	var allErrs field.ErrorList
	specPath := field.NewPath("spec")
	if strings.TrimSpace(appEnv.Spec.App.Image) == "" {
		allErrs = append(allErrs, field.Required(specPath.Child("app", "image"), "image is required"))
	}
	if appEnv.Spec.App.Replicas != nil && *appEnv.Spec.App.Replicas < 0 {
		allErrs = append(allErrs, field.Invalid(specPath.Child("app", "replicas"), *appEnv.Spec.App.Replicas, "must be >= 0"))
	}
	if appEnv.Spec.Autoscaling.Enabled {
		if appEnv.Spec.Autoscaling.MaxReplicas <= 0 {
			allErrs = append(allErrs, field.Invalid(specPath.Child("autoscaling", "maxReplicas"), appEnv.Spec.Autoscaling.MaxReplicas, "must be > 0"))
		}
		if appEnv.Spec.Autoscaling.MinReplicas != nil && *appEnv.Spec.Autoscaling.MinReplicas > appEnv.Spec.Autoscaling.MaxReplicas {
			allErrs = append(allErrs, field.Invalid(specPath.Child("autoscaling", "minReplicas"), *appEnv.Spec.Autoscaling.MinReplicas, "must be <= maxReplicas"))
		}
	}
	if appEnv.Spec.Ingress.Enabled {
		if !hostRegex.MatchString(appEnv.Spec.Ingress.Host) {
			allErrs = append(allErrs, field.Invalid(specPath.Child("ingress", "host"), appEnv.Spec.Ingress.Host, "must be a valid RFC 1123 hostname"))
		}
		if err := v.ensureUniqueHost(ctx, appEnv); err != nil {
			allErrs = append(allErrs, field.Invalid(specPath.Child("ingress", "host"), appEnv.Spec.Ingress.Host, err.Error()))
		}
	}
	if appEnv.Spec.Migration.Enabled {
		if !appEnv.Spec.Database.Enabled {
			allErrs = append(allErrs, field.Invalid(specPath.Child("migration", "enabled"), true, "database must be enabled"))
		}
		if appEnv.Spec.Migration.Image == "" || appEnv.Spec.Migration.MigrationID == "" {
			allErrs = append(allErrs, field.Required(specPath.Child("migration"), "migration.image and migration.migrationID are required"))
		}
	}
	if appEnv.Spec.Restore.Enabled && (appEnv.Spec.Restore.Image == "" || appEnv.Spec.Restore.Source == "" || appEnv.Spec.Restore.TriggerNonce == "") {
		allErrs = append(allErrs, field.Required(specPath.Child("restore"), "restore.image, restore.source, and restore.triggerNonce are required"))
	}
	if err := validateResourceBounds(specPath.Child("app", "resources"), appEnv.Spec.App.Resources); err != nil {
		allErrs = append(allErrs, err...)
	}
	if appEnv.Spec.Security.PodDisruptionBudget.MinAvailable != nil && appEnv.Spec.Security.PodDisruptionBudget.MaxUnavailable != nil {
		allErrs = append(allErrs, field.Invalid(specPath.Child("security", "podDisruptionBudget"), "both set", "only one of minAvailable or maxUnavailable may be set"))
	}
	for _, ref := range appEnv.Spec.App.SecretRefs {
		if strings.Contains(ref.Name, "/") {
			allErrs = append(allErrs, field.Invalid(specPath.Child("app", "secretRefs"), ref.Name, "cross-namespace secret references are not allowed"))
		}
		if ref.MountAs == "volume" && ref.MountPath == "" {
			allErrs = append(allErrs, field.Required(specPath.Child("app", "secretRefs"), "mountPath is required when mountAs=volume"))
		}
	}
	if strings.Contains(appEnv.Spec.Database.SecretRef, "/") {
		allErrs = append(allErrs, field.Invalid(specPath.Child("database", "secretRef"), appEnv.Spec.Database.SecretRef, "cross-namespace secret references are not allowed"))
	}
	if old != nil {
		if old.Spec.Database.Mode != "" && old.Spec.Database.Mode != appEnv.Spec.Database.Mode {
			allErrs = append(allErrs, field.Forbidden(specPath.Child("database", "mode"), "field is immutable"))
		}
		if old.Spec.Ingress.Host != "" && old.Spec.Ingress.Host != appEnv.Spec.Ingress.Host {
			allErrs = append(allErrs, field.Forbidden(specPath.Child("ingress", "host"), "field is immutable once ingress exists"))
		}
	}
	if len(allErrs) > 0 {
		return apierrors.NewInvalid(appsv1beta1.GroupVersion.WithKind("AppEnvironment").GroupKind(), appEnv.Name, allErrs)
	}
	return nil
}

func validateResourceBounds(path *field.Path, resources corev1.ResourceRequirements) field.ErrorList {
	var errs field.ErrorList
	for resourceName, request := range resources.Requests {
		if limit, ok := resources.Limits[resourceName]; ok && limit.Cmp(request) < 0 {
			errs = append(errs, field.Invalid(path.Child("limits"), limit.String(), "limit must be >= request"))
		}
	}
	return errs
}

func (v *AppEnvironmentValidator) ensureUniqueHost(ctx context.Context, appEnv *appsv1beta1.AppEnvironment) error {
	var list appsv1beta1.AppEnvironmentList
	if err := v.Client.List(ctx, &list); err != nil {
		return err
	}
	for _, item := range list.Items {
		if item.Namespace == appEnv.Namespace && item.Name == appEnv.Name {
			continue
		}
		if item.Spec.Ingress.Host == appEnv.Spec.Ingress.Host {
			return fmt.Errorf("host %s is already claimed by %s/%s", appEnv.Spec.Ingress.Host, item.Namespace, item.Name)
		}
	}
	return nil
}

var _ webhook.CustomValidator = &AppEnvironmentValidator{}

func setupValidator(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&appsv1beta1.AppEnvironment{}).WithValidator(&AppEnvironmentValidator{Client: mgr.GetClient()}).Complete()
}
