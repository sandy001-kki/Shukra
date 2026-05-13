package cleanup

import (
	"context"
	"encoding/json"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type BackupHook struct {
	client client.Client
}

func NewBackupHook(c client.Client) Hook {
	return &BackupHook{client: c}
}

func (h *BackupHook) Name() string {
	return "backup-metadata-cleanup"
}

func (h *BackupHook) Cleanup(ctx context.Context, env CleanupTarget) error {
	if env.BackupDestination == "" {
		ctrl.LoggerFrom(ctx).Info("backup cleanup skipped; no backup destination configured")
		return nil
	}

	hookCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	log := ctrl.LoggerFrom(ctx).WithValues("hook", h.Name(), "destination", env.BackupDestination)
	if err := h.archiveBackupMetadata(hookCtx, env); err != nil {
		log.Error(err, "backup metadata store unreachable; treating cleanup as best-effort")
		return nil
	}

	cronJob := &batchv1.CronJob{}
	err := h.client.Get(hookCtx, types.NamespacedName{Name: env.Name + "-backup", Namespace: env.Namespace}, cronJob)
	if apierrors.IsNotFound(err) {
		log.Info("backup metadata archived; backup CronJob already gone")
		return nil
	}
	if err != nil {
		log.Error(err, "backup destination or CronJob unreachable; treating cleanup as best-effort")
		return nil
	}
	if err := h.client.Delete(hookCtx, cronJob); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	log.Info("backup metadata archived and CronJob delete requested")
	return nil
}

func (h *BackupHook) archiveBackupMetadata(ctx context.Context, env CleanupTarget) error {
	record := map[string]string{
		"name":          env.Name,
		"namespace":     env.Namespace,
		"destination":   env.BackupDestination,
		"status":        "archived",
		"archivedAtUTC": time.Now().UTC().Format(time.RFC3339),
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shukra-backup-metadata",
			Namespace: env.Namespace,
		},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, h.client, cm, func() error {
		if cm.Labels == nil {
			cm.Labels = map[string]string{}
		}
		cm.Labels["app.kubernetes.io/managed-by"] = "shukra-operator"
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}
		cm.Data[env.Name] = string(payload)
		return nil
	})
	return err
}
