// This file contains the main AppEnvironment reconciler. It exists to implement
// the ordered platform reconciliation flow, error classification, status
// management, event emission, and controller-runtime setup.
package controllers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
	"github.com/sandy001-kki/Shukra/internal/finalizer"
	"github.com/sandy001-kki/Shukra/internal/resources"
	statusutil "github.com/sandy001-kki/Shukra/internal/status"
	"github.com/sandy001-kki/Shukra/pkg/events"
	shukrametrics "github.com/sandy001-kki/Shukra/pkg/metrics"
)

// Secrets are read-only because the operator references existing secrets and
// never creates, mutates, or deletes them.
// Events only need create;patch because event writers append lifecycle records.
// Leases are required for controller-runtime leader election in HA deployments.
// +kubebuilder:rbac:groups=apps.shukra.io,resources=appenvironments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.shukra.io,resources=appenvironments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.shukra.io,resources=appenvironments/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services;configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses;networkpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs;cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
type AppEnvironmentReconciler struct {
	client.Client
	Scheme                  *runtime.Scheme
	EventRecorder           recordLike
	MaxConcurrentReconciles int
	WatchNamespace          string
}

type recordLike interface {
	Event(object runtime.Object, eventtype, reason, message string)
	Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{})
}

type permanentError struct{ err error }

func (e permanentError) Error() string { return e.err.Error() }

func retryOnConflict(fn func() error) error {
	return retry.RetryOnConflict(retry.DefaultRetry, fn)
}

func (r *AppEnvironmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	start := time.Now()
	result := "success"
	reconcileID := string(req.NamespacedName.Namespace) + "-" + req.NamespacedName.Name + "-" + time.Now().UTC().Format("20060102150405")
	log := ctrl.LoggerFrom(ctx).WithValues("namespace", req.Namespace, "name", req.Name, "reconcileID", reconcileID)
	defer func() {
		shukrametrics.ReconcileDuration.WithLabelValues(req.Namespace, result).Observe(time.Since(start).Seconds())
	}()

	appEnv := &appsv1beta1.AppEnvironment{}
	if err := r.Get(ctx, req.NamespacedName, appEnv); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		result = "error"
		return ctrl.Result{}, err
	}
	log = log.WithValues("generation", appEnv.Generation)
	eventRecorder := events.New(r.EventRecorder)
	eventRecorder.RecordReconcileStarted(appEnv)

	if !appEnv.DeletionTimestamp.IsZero() {
		eventRecorder.RecordDeletionStarted(appEnv)
		return finalizer.HandleDeletion(ctx, r.Client, appEnv, log)
	}
	if err := finalizer.Ensure(ctx, r.Client, appEnv); err != nil {
		result = "error"
		return ctrl.Result{}, err
	}

	now := metav1.Now()
	appEnv.Status.ObservedGeneration = appEnv.Generation
	appEnv.Status.LastAppliedSpecHash = hashSpec(appEnv.Spec)

	if appEnv.Spec.Paused {
		statusutil.SetCondition(&appEnv.Status.Conditions, appsv1beta1.ConditionPaused, metav1.ConditionTrue, "Paused", "Reconciliation paused by spec", appEnv.Generation)
		statusutil.SetCondition(&appEnv.Status.Conditions, appsv1beta1.ConditionReady, metav1.ConditionFalse, "Paused", "Paused resources are not mutated", appEnv.Generation)
		appEnv.Status.Phase = appsv1beta1.PhasePaused
		appEnv.Status.LastSuccessfulReconcileTime = &now
		eventRecorder.RecordPaused(appEnv)
		return ctrl.Result{}, r.persistStatus(ctx, appEnv)
	}
	statusutil.SetCondition(&appEnv.Status.Conditions, appsv1beta1.ConditionPaused, metav1.ConditionFalse, "Running", "Reconciliation active", appEnv.Generation)
	eventRecorder.RecordUnpaused(appEnv)

	if err := r.validateSpecDependencies(ctx, appEnv); err != nil {
		statusutil.SetCondition(&appEnv.Status.Conditions, appsv1beta1.ConditionSpecValid, metav1.ConditionFalse, "ValidationFailed", err.Error(), appEnv.Generation)
		appEnv.Status.Phase = appsv1beta1.PhaseFailed
		appEnv.Status.LastError = err.Error()
		appEnv.Status.FailureCount++
		result = "error"
		shukrametrics.ReconcileFailures.WithLabelValues(appEnv.Namespace, "validation").Inc()
		_ = r.persistStatus(ctx, appEnv)
		return ctrl.Result{}, permanentError{err: err}
	}
	statusutil.SetCondition(&appEnv.Status.Conditions, appsv1beta1.ConditionSpecValid, metav1.ConditionTrue, "Validated", "Specification is valid", appEnv.Generation)

	steps := []func(context.Context, *appsv1beta1.AppEnvironment, events.Recorder, logr.Logger) error{
		r.reconcileConfigMap,
		r.validateSecretReferences,
		r.reconcileService,
		r.reconcileDeployment,
		r.reconcileHPA,
		r.reconcileMigrationJob,
		r.reconcileRestoreJob,
		r.reconcileIngress,
		r.reconcileNetworkPolicy,
		r.reconcilePDB,
		r.reconcileBackupCronJob,
	}
	for _, step := range steps {
		if err := step(ctx, appEnv, eventRecorder, log); err != nil {
			var permanent permanentError
			if errors.As(err, &permanent) {
				statusutil.SetCondition(&appEnv.Status.Conditions, appsv1beta1.ConditionReady, metav1.ConditionFalse, "PermanentError", err.Error(), appEnv.Generation)
				appEnv.Status.Phase = appsv1beta1.PhaseFailed
				appEnv.Status.LastError = err.Error()
				_ = r.persistStatus(ctx, appEnv)
				return ctrl.Result{}, nil
			}
			result = "error"
			appEnv.Status.FailureCount++
			appEnv.Status.LastError = err.Error()
			statusutil.SetCondition(&appEnv.Status.Conditions, appsv1beta1.ConditionReady, metav1.ConditionFalse, "TransientError", err.Error(), appEnv.Generation)
			_ = r.persistStatus(ctx, appEnv)
			shukrametrics.ReconcileFailures.WithLabelValues(appEnv.Namespace, "transient").Inc()
			return ctrl.Result{}, err
		}
	}

	appEnv.Status.LastError = ""
	appEnv.Status.FailureCount = 0
	statusutil.SetCondition(&appEnv.Status.Conditions, appsv1beta1.ConditionReady, metav1.ConditionTrue, "Ready", "All resources reconciled", appEnv.Generation)
	appEnv.Status.Phase = statusutil.ComputePhase(appEnv.Status.Conditions, false, appEnv.Spec.Restore.Enabled && appEnv.Status.LastProcessedRestoreNonce != appEnv.Spec.Restore.TriggerNonce)
	appEnv.Status.LastSuccessfulReconcileTime = &now
	return ctrl.Result{}, r.persistStatus(ctx, appEnv)
}

func (r *AppEnvironmentReconciler) persistStatus(ctx context.Context, appEnv *appsv1beta1.AppEnvironment) error {
	desired := appEnv.DeepCopy().Status
	key := types.NamespacedName{Name: appEnv.Name, Namespace: appEnv.Namespace}
	return retryOnConflict(func() error {
		current := &appsv1beta1.AppEnvironment{}
		if err := r.Get(ctx, key, current); err != nil {
			return err
		}
		current.Status = desired
		return r.Status().Update(ctx, current)
	})
}

func (r *AppEnvironmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.MaxConcurrentReconciles == 0 {
		r.MaxConcurrentReconciles = 5
	}
	rateLimiter := workqueue.NewItemExponentialFailureRateLimiter(5*time.Second, 10*time.Minute)
	// Generation predicates suppress status-only updates on the primary object,
	// and resource version filtering on child objects avoids noisy reconcile loops
	// from status churn, which reduces API server load in large clusters.
	childPredicate := predicate.ResourceVersionChangedPredicate{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1beta1.AppEnvironment{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&appsv1.Deployment{}, builder.WithPredicates(childPredicate)).
		Owns(&corev1.Service{}, builder.WithPredicates(childPredicate)).
		Owns(&corev1.ConfigMap{}, builder.WithPredicates(childPredicate)).
		Owns(&autoscalingv2.HorizontalPodAutoscaler{}, builder.WithPredicates(childPredicate)).
		Owns(&networkingv1.Ingress{}, builder.WithPredicates(childPredicate)).
		Owns(&networkingv1.NetworkPolicy{}, builder.WithPredicates(childPredicate)).
		Owns(&policyv1.PodDisruptionBudget{}, builder.WithPredicates(childPredicate)).
		Owns(&batchv1.Job{}, builder.WithPredicates(childPredicate)).
		Owns(&batchv1.CronJob{}, builder.WithPredicates(childPredicate)).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: r.MaxConcurrentReconciles,
			RateLimiter:             rateLimiter,
		}).
		Complete(r)
}

func (r *AppEnvironmentReconciler) validateSpecDependencies(ctx context.Context, appEnv *appsv1beta1.AppEnvironment) error {
	if appEnv.Spec.Migration.Enabled && !appEnv.Spec.Database.Enabled {
		return fmt.Errorf("migration enabled requires database enabled")
	}
	if appEnv.Spec.Restore.Enabled && (appEnv.Spec.Restore.Source == "" || appEnv.Spec.Restore.Image == "") {
		return fmt.Errorf("restore requires source and image")
	}
	if appEnv.Spec.Autoscaling.Enabled && appEnv.Spec.Autoscaling.MinReplicas != nil && *appEnv.Spec.Autoscaling.MinReplicas > appEnv.Spec.Autoscaling.MaxReplicas {
		return fmt.Errorf("autoscaling minReplicas cannot exceed maxReplicas")
	}
	if err := r.validateSecretRefs(ctx, appEnv); err != nil {
		return err
	}
	if appEnv.Spec.Ingress.Enabled {
		var list appsv1beta1.AppEnvironmentList
		if err := r.List(ctx, &list); err != nil {
			return err
		}
		for _, item := range list.Items {
			if item.UID != appEnv.UID && item.Spec.Ingress.Host == appEnv.Spec.Ingress.Host {
				return fmt.Errorf("ingress host %q is already in use", appEnv.Spec.Ingress.Host)
			}
		}
	}
	return nil
}

func (r *AppEnvironmentReconciler) validateSecretRefs(ctx context.Context, appEnv *appsv1beta1.AppEnvironment) error {
	for _, ref := range appEnv.Spec.App.SecretRefs {
		secret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: appEnv.Namespace}, secret); err != nil {
			return fmt.Errorf("referenced secret %q not found in namespace %q: %w", ref.Name, appEnv.Namespace, err)
		}
	}
	if appEnv.Spec.Database.SecretRef != "" {
		secret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Name: appEnv.Spec.Database.SecretRef, Namespace: appEnv.Namespace}, secret); err != nil {
			return fmt.Errorf("database secret %q not found in namespace %q: %w", appEnv.Spec.Database.SecretRef, appEnv.Namespace, err)
		}
	}
	return nil
}

func hashSpec(spec any) string {
	payload, err := json.Marshal(spec)
	utilruntime.Must(err)
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func (r *AppEnvironmentReconciler) reconcileConfigMap(ctx context.Context, appEnv *appsv1beta1.AppEnvironment, recorder events.Recorder, _ logr.Logger) error {
	cm := resources.ConfigMap(appEnv)
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		desired := resources.ConfigMap(appEnv)
		cm.Labels = desired.Labels
		cm.Annotations = desired.Annotations
		cm.Data = desired.Data
		return controllerutil.SetControllerReference(appEnv, cm, r.Scheme)
	})
	if err != nil {
		err = retryOnConflict(func() error {
			_, retryErr := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
				desired := resources.ConfigMap(appEnv)
				cm.Labels = desired.Labels
				cm.Annotations = desired.Annotations
				cm.Data = desired.Data
				return controllerutil.SetControllerReference(appEnv, cm, r.Scheme)
			})
			return retryErr
		})
		if err != nil {
			return err
		}
	}
	appEnv.Status.ChildResources.ConfigMapName = cm.Name
	statusutil.SetCondition(&appEnv.Status.Conditions, appsv1beta1.ConditionConfigReady, metav1.ConditionTrue, "Configured", "ConfigMap reconciled", appEnv.Generation)
	recorder.RecordChildReconciled(appEnv, "ConfigMap")
	return nil
}

func (r *AppEnvironmentReconciler) validateSecretReferences(ctx context.Context, appEnv *appsv1beta1.AppEnvironment, _ events.Recorder, _ logr.Logger) error {
	return r.validateSecretRefs(ctx, appEnv)
}

func (r *AppEnvironmentReconciler) reconcileService(ctx context.Context, appEnv *appsv1beta1.AppEnvironment, recorder events.Recorder, _ logr.Logger) error {
	if !appEnv.EffectiveServiceEnabled() {
		statusutil.SetCondition(&appEnv.Status.Conditions, appsv1beta1.ConditionServiceReady, metav1.ConditionTrue, "Skipped", "Service disabled", appEnv.Generation)
		return nil
	}
	svc := resources.Service(appEnv)
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		desired := resources.Service(appEnv)
		svc.Labels = desired.Labels
		svc.Annotations = desired.Annotations
		svc.Spec = desired.Spec
		return controllerutil.SetControllerReference(appEnv, svc, r.Scheme)
	})
	if err != nil {
		err = retryOnConflict(func() error {
			_, retryErr := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
				desired := resources.Service(appEnv)
				svc.Labels = desired.Labels
				svc.Annotations = desired.Annotations
				svc.Spec = desired.Spec
				return controllerutil.SetControllerReference(appEnv, svc, r.Scheme)
			})
			return retryErr
		})
		if err != nil {
			return err
		}
	}
	appEnv.Status.ChildResources.ServiceName = svc.Name
	statusutil.SetCondition(&appEnv.Status.Conditions, appsv1beta1.ConditionServiceReady, metav1.ConditionTrue, "Ready", "Service reconciled", appEnv.Generation)
	recorder.RecordChildReconciled(appEnv, "Service")
	return nil
}

func (r *AppEnvironmentReconciler) reconcileDeployment(ctx context.Context, appEnv *appsv1beta1.AppEnvironment, recorder events.Recorder, _ logr.Logger) error {
	deployment := resources.Deployment(appEnv)
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		desired := resources.Deployment(appEnv)
		deployment.Labels = desired.Labels
		deployment.Spec = desired.Spec
		return controllerutil.SetControllerReference(appEnv, deployment, r.Scheme)
	})
	if err != nil {
		err = retryOnConflict(func() error {
			_, retryErr := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
				desired := resources.Deployment(appEnv)
				deployment.Labels = desired.Labels
				deployment.Spec = desired.Spec
				return controllerutil.SetControllerReference(appEnv, deployment, r.Scheme)
			})
			return retryErr
		})
		if err != nil {
			return err
		}
	}
	appEnv.Status.ChildResources.DeploymentName = deployment.Name
	statusutil.SetCondition(&appEnv.Status.Conditions, appsv1beta1.ConditionDeploymentReady, metav1.ConditionTrue, "Ready", "Deployment reconciled", appEnv.Generation)
	recorder.RecordChildReconciled(appEnv, "Deployment")
	return nil
}

func (r *AppEnvironmentReconciler) reconcileHPA(ctx context.Context, appEnv *appsv1beta1.AppEnvironment, recorder events.Recorder, _ logr.Logger) error {
	if !appEnv.Spec.Autoscaling.Enabled {
		return nil
	}
	hpa := resources.HPA(appEnv)
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, hpa, func() error {
		desired := resources.HPA(appEnv)
		hpa.Labels = desired.Labels
		hpa.Spec = desired.Spec
		return controllerutil.SetControllerReference(appEnv, hpa, r.Scheme)
	})
	if err != nil {
		err = retryOnConflict(func() error {
			_, retryErr := controllerutil.CreateOrUpdate(ctx, r.Client, hpa, func() error {
				desired := resources.HPA(appEnv)
				hpa.Labels = desired.Labels
				hpa.Spec = desired.Spec
				return controllerutil.SetControllerReference(appEnv, hpa, r.Scheme)
			})
			return retryErr
		})
		if err != nil {
			return err
		}
	}
	appEnv.Status.ChildResources.HPAName = hpa.Name
	recorder.RecordChildReconciled(appEnv, "HorizontalPodAutoscaler")
	return nil
}

func (r *AppEnvironmentReconciler) reconcileMigrationJob(ctx context.Context, appEnv *appsv1beta1.AppEnvironment, recorder events.Recorder, _ logr.Logger) error {
	if !appEnv.Spec.Migration.Enabled {
		statusutil.SetCondition(&appEnv.Status.Conditions, appsv1beta1.ConditionMigrationReady, metav1.ConditionTrue, "Skipped", "Migration disabled", appEnv.Generation)
		return nil
	}
	jobName := appEnv.MigrationJobName()
	job := &batchv1.Job{}
	err := r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: appEnv.Namespace}, job)
	if apierrors.IsNotFound(err) && appEnv.Status.LastAppliedMigrationID != appEnv.Spec.Migration.MigrationID {
		job = resources.MigrationJob(appEnv)
		if err := controllerutil.SetControllerReference(appEnv, job, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(ctx, job); err != nil {
			return err
		}
		appEnv.Status.LastAppliedMigrationID = appEnv.Spec.Migration.MigrationID
		appEnv.Status.ChildResources.MigrationJobName = job.Name
		statusutil.SetCondition(&appEnv.Status.Conditions, appsv1beta1.ConditionMigrationReady, metav1.ConditionTrue, "Started", "Migration job created", appEnv.Generation)
		recorder.RecordMigrationStarted(appEnv)
		recorder.RecordChildReconciled(appEnv, "MigrationJob")
		shukrametrics.MigrationsTotal.WithLabelValues(appEnv.Namespace, "success").Inc()
		return nil
	}
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	statusutil.SetCondition(&appEnv.Status.Conditions, appsv1beta1.ConditionMigrationReady, metav1.ConditionTrue, "Noop", "Migration already processed", appEnv.Generation)
	return nil
}

func (r *AppEnvironmentReconciler) reconcileRestoreJob(ctx context.Context, appEnv *appsv1beta1.AppEnvironment, recorder events.Recorder, _ logr.Logger) error {
	if !appEnv.Spec.Restore.Enabled {
		statusutil.SetCondition(&appEnv.Status.Conditions, appsv1beta1.ConditionRestoreReady, metav1.ConditionTrue, "Skipped", "Restore disabled", appEnv.Generation)
		return nil
	}
	if appEnv.Status.LastProcessedRestoreNonce == appEnv.Spec.Restore.TriggerNonce {
		statusutil.SetCondition(&appEnv.Status.Conditions, appsv1beta1.ConditionRestoreReady, metav1.ConditionTrue, "Noop", "Restore nonce already processed", appEnv.Generation)
		return nil
	}
	job := resources.RestoreJob(appEnv)
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, job, func() error {
		return controllerutil.SetControllerReference(appEnv, job, r.Scheme)
	})
	if err != nil {
		err = retryOnConflict(func() error {
			_, retryErr := controllerutil.CreateOrUpdate(ctx, r.Client, job, func() error {
				return controllerutil.SetControllerReference(appEnv, job, r.Scheme)
			})
			return retryErr
		})
		if err != nil {
			return err
		}
	}
	appEnv.Status.Phase = appsv1beta1.PhaseRestoring
	appEnv.Status.LastProcessedRestoreNonce = appEnv.Spec.Restore.TriggerNonce
	appEnv.Status.ChildResources.RestoreJobName = job.Name
	statusutil.SetCondition(&appEnv.Status.Conditions, appsv1beta1.ConditionRestoreReady, metav1.ConditionTrue, "Started", "Restore job created", appEnv.Generation)
	recorder.RecordRestoreStarted(appEnv)
	recorder.RecordChildReconciled(appEnv, "RestoreJob")
	shukrametrics.RestoresTotal.WithLabelValues(appEnv.Namespace, "success").Inc()
	return nil
}

func (r *AppEnvironmentReconciler) reconcileIngress(ctx context.Context, appEnv *appsv1beta1.AppEnvironment, recorder events.Recorder, _ logr.Logger) error {
	if !appEnv.Spec.Ingress.Enabled {
		statusutil.SetCondition(&appEnv.Status.Conditions, appsv1beta1.ConditionIngressReady, metav1.ConditionTrue, "Skipped", "Ingress disabled", appEnv.Generation)
		return nil
	}
	ingress := resources.Ingress(appEnv)
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, ingress, func() error {
		desired := resources.Ingress(appEnv)
		ingress.Labels = desired.Labels
		ingress.Annotations = desired.Annotations
		ingress.Spec = desired.Spec
		return controllerutil.SetControllerReference(appEnv, ingress, r.Scheme)
	})
	if err != nil {
		err = retryOnConflict(func() error {
			_, retryErr := controllerutil.CreateOrUpdate(ctx, r.Client, ingress, func() error {
				desired := resources.Ingress(appEnv)
				ingress.Labels = desired.Labels
				ingress.Annotations = desired.Annotations
				ingress.Spec = desired.Spec
				return controllerutil.SetControllerReference(appEnv, ingress, r.Scheme)
			})
			return retryErr
		})
		if err != nil {
			return err
		}
	}
	appEnv.Status.ChildResources.IngressName = ingress.Name
	appEnv.Status.URL = "https://" + appEnv.Spec.Ingress.Host + appEnv.Spec.Ingress.Path
	statusutil.SetCondition(&appEnv.Status.Conditions, appsv1beta1.ConditionIngressReady, metav1.ConditionTrue, "Ready", "Ingress reconciled", appEnv.Generation)
	recorder.RecordChildReconciled(appEnv, "Ingress")
	return nil
}

func (r *AppEnvironmentReconciler) reconcileNetworkPolicy(ctx context.Context, appEnv *appsv1beta1.AppEnvironment, recorder events.Recorder, _ logr.Logger) error {
	if len(appEnv.Spec.Security.NetworkPolicy.IngressRules) == 0 && len(appEnv.Spec.Security.NetworkPolicy.EgressRules) == 0 {
		return nil
	}
	netpol := resources.NetworkPolicy(appEnv)
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, netpol, func() error {
		desired := resources.NetworkPolicy(appEnv)
		netpol.Labels = desired.Labels
		netpol.Spec = desired.Spec
		return controllerutil.SetControllerReference(appEnv, netpol, r.Scheme)
	})
	if err != nil {
		err = retryOnConflict(func() error {
			_, retryErr := controllerutil.CreateOrUpdate(ctx, r.Client, netpol, func() error {
				desired := resources.NetworkPolicy(appEnv)
				netpol.Labels = desired.Labels
				netpol.Spec = desired.Spec
				return controllerutil.SetControllerReference(appEnv, netpol, r.Scheme)
			})
			return retryErr
		})
		if err != nil {
			return err
		}
	}
	appEnv.Status.ChildResources.NetworkPolicyName = netpol.Name
	recorder.RecordChildReconciled(appEnv, "NetworkPolicy")
	return nil
}

func (r *AppEnvironmentReconciler) reconcilePDB(ctx context.Context, appEnv *appsv1beta1.AppEnvironment, recorder events.Recorder, _ logr.Logger) error {
	if !appEnv.Spec.Security.PodDisruptionBudget.Enabled {
		return nil
	}
	pdb, err := resources.PDB(appEnv)
	if err != nil {
		return permanentError{err: err}
	}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, pdb, func() error {
		desired, buildErr := resources.PDB(appEnv)
		if buildErr != nil {
			return buildErr
		}
		pdb.Labels = desired.Labels
		pdb.Spec = desired.Spec
		return controllerutil.SetControllerReference(appEnv, pdb, r.Scheme)
	})
	if err != nil {
		err = retryOnConflict(func() error {
			_, retryErr := controllerutil.CreateOrUpdate(ctx, r.Client, pdb, func() error {
				desired, buildErr := resources.PDB(appEnv)
				if buildErr != nil {
					return buildErr
				}
				pdb.Labels = desired.Labels
				pdb.Spec = desired.Spec
				return controllerutil.SetControllerReference(appEnv, pdb, r.Scheme)
			})
			return retryErr
		})
		if err != nil {
			return err
		}
	}
	appEnv.Status.ChildResources.PDBName = pdb.Name
	recorder.RecordChildReconciled(appEnv, "PodDisruptionBudget")
	return nil
}

func (r *AppEnvironmentReconciler) reconcileBackupCronJob(ctx context.Context, appEnv *appsv1beta1.AppEnvironment, recorder events.Recorder, _ logr.Logger) error {
	if !appEnv.Spec.Backup.Enabled {
		shukrametrics.BackupConfigured.WithLabelValues(appEnv.Namespace).Set(0)
		statusutil.SetCondition(&appEnv.Status.Conditions, appsv1beta1.ConditionBackupReady, metav1.ConditionTrue, "Skipped", "Backup disabled", appEnv.Generation)
		return nil
	}
	cronjob := resources.BackupCronJob(appEnv)
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cronjob, func() error {
		desired := resources.BackupCronJob(appEnv)
		cronjob.Labels = desired.Labels
		cronjob.Spec = desired.Spec
		return controllerutil.SetControllerReference(appEnv, cronjob, r.Scheme)
	})
	if err != nil {
		err = retryOnConflict(func() error {
			_, retryErr := controllerutil.CreateOrUpdate(ctx, r.Client, cronjob, func() error {
				desired := resources.BackupCronJob(appEnv)
				cronjob.Labels = desired.Labels
				cronjob.Spec = desired.Spec
				return controllerutil.SetControllerReference(appEnv, cronjob, r.Scheme)
			})
			return retryErr
		})
		if err != nil {
			return err
		}
	}
	appEnv.Status.ChildResources.BackupCronJobName = cronjob.Name
	statusutil.SetCondition(&appEnv.Status.Conditions, appsv1beta1.ConditionBackupReady, metav1.ConditionTrue, "Ready", "Backup CronJob reconciled", appEnv.Generation)
	recorder.RecordChildReconciled(appEnv, "BackupCronJob")
	shukrametrics.BackupConfigured.WithLabelValues(appEnv.Namespace).Set(1)
	return nil
}
