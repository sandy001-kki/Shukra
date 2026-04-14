// This file owns finalizer management and deletion cleanup hooks. It exists to
// keep deletion behavior separate from steady-state reconciliation logic.
package finalizer

import (
	"context"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
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

	// MOCK: In production, replace this with a real database client call.
	log.Info("would drop schema", "schema", appEnv.Spec.Database.SchemaName)
	// MOCK: In production, replace this with a real backup client call.
	log.Info("would delete backup records for application", "destination", appEnv.Spec.Backup.Destination)
	// MOCK: In production, replace this with a real DNS client call.
	log.Info("would remove DNS record", "host", appEnv.Spec.Ingress.Host)

	controllerutil.RemoveFinalizer(appEnv, Name)
	return ctrl.Result{}, c.Update(ctx, appEnv)
}
