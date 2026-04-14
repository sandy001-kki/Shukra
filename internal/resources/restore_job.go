// This file builds restore Jobs. It exists because restore operations are
// explicitly nonce-driven and must be modeled separately from migrations.
package resources

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
)

func RestoreJob(appEnv *appsv1beta1.AppEnvironment) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appEnv.RestoreJobName(),
			Namespace: appEnv.Namespace,
			Labels:    appEnv.Labels("restore"),
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: appEnv.Labels("restore")},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{{
						Name:            "restore",
						Image:           appEnv.Spec.Restore.Image,
						ImagePullPolicy: appEnv.Spec.App.ImagePullPolicy,
						Command: []string{"/bin/sh", "-c"},
						Args: []string{appEnv.Spec.Restore.Source},
						Resources:       appEnv.Spec.App.Resources,
					}},
				},
			},
		},
	}
}
