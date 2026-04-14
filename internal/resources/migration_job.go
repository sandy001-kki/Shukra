// This file builds migration Jobs. It exists because migration idempotency and
// job configuration have rules distinct from other child resources.
package resources

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
)

func MigrationJob(appEnv *appsv1beta1.AppEnvironment) *batchv1.Job {
	backoff := int32(3)
	if appEnv.Spec.Migration.BackoffLimit != nil {
		backoff = *appEnv.Spec.Migration.BackoffLimit
	}
	deadline := int64(300)
	if appEnv.Spec.Migration.ActiveDeadlineSeconds != nil {
		deadline = *appEnv.Spec.Migration.ActiveDeadlineSeconds
	}
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appEnv.MigrationJobName(),
			Namespace: appEnv.Namespace,
			Labels:    appEnv.Labels("migration"),
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:          &backoff,
			ActiveDeadlineSeconds: &deadline,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: appEnv.Labels("migration")},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{{
						Name:            "migration",
						Image:           appEnv.Spec.Migration.Image,
						ImagePullPolicy: appEnv.Spec.App.ImagePullPolicy,
						Command:         appEnv.Spec.Migration.Command,
						Args:            appEnv.Spec.Migration.Args,
						EnvFrom:         buildEnvFrom(appEnv),
						Resources:       appEnv.Spec.App.Resources,
					}},
				},
			},
		},
	}
}
