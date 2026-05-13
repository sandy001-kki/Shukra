// This file owns finalizer management and deletion cleanup hooks. It exists to
// keep deletion behavior separate from steady-state reconciliation logic.
package finalizer

import (
	"context"
	"errors"
	"time"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
	"github.com/sandy001-kki/Shukra/internal/cleanup"
	"github.com/sandy001-kki/Shukra/internal/shadow"
)

const Name = "apps.shukra.io/finalizer"

func Ensure(ctx context.Context, c client.Client, appEnv *appsv1beta1.AppEnvironment) error {
	if controllerutil.ContainsFinalizer(appEnv, Name) {
		return nil
	}
	controllerutil.AddFinalizer(appEnv, Name)
	return c.Update(ctx, appEnv)
}

func HandleDeletion(ctx context.Context, c client.Client, appEnv *appsv1beta1.AppEnvironment, log logr.Logger) (ctrl.Result, error) {
	appEnv.Status.Phase = appsv1beta1.PhaseDeleting
	if !controllerutil.ContainsFinalizer(appEnv, Name) {
		return ctrl.Result{}, nil
	}

	if shadow.IsShadow(appEnv) {
		log.Info("skipping external cleanup hooks for shadow environment")
		controllerutil.RemoveFinalizer(appEnv, Name)
		return ctrl.Result{}, c.Update(ctx, appEnv)
	}

	target := cleanup.CleanupTarget{
		Name:              appEnv.Name,
		Namespace:         appEnv.Namespace,
		DatabaseMode:      appEnv.Spec.Database.Mode,
		DatabaseSecret:    appEnv.Spec.Database.SecretRef,
		BackupDestination: appEnv.Spec.Backup.Destination,
		IngressHost:       appEnv.Spec.Ingress.Host,
	}
	hooks := []cleanup.Hook{
		cleanup.NewDatabaseHook(c),
		cleanup.NewBackupHook(c),
		cleanup.NewDNSHook(c),
	}
	for _, hook := range hooks {
		hookCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		err := hook.Cleanup(hookCtx, target)
		cancel()
		if err == nil {
			log.Info("cleanup hook completed", "hook", hook.Name())
			continue
		}
		if errors.Is(err, cleanup.ErrTransient) {
			log.Error(err, "cleanup hook requested retry", "hook", hook.Name())
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
		log.Error(err, "cleanup hook failed, continuing", "hook", hook.Name())
	}

	controllerutil.RemoveFinalizer(appEnv, Name)
	return ctrl.Result{}, c.Update(ctx, appEnv)
}
