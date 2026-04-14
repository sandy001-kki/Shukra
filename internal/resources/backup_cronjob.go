// This file builds backup CronJobs. It exists so backup scheduling details stay
// separate from controller orchestration and status/error handling.
package resources

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
)

func BackupCronJob(appEnv *appsv1beta1.AppEnvironment) *batchv1.CronJob {
	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Name(appEnv, "backup"),
			Namespace: appEnv.Namespace,
			Labels:    appEnv.Labels("backup"),
		},
		Spec: batchv1.CronJobSpec{
			Schedule: appEnv.Spec.Backup.Schedule,
			Suspend:  &appEnv.Spec.Backup.Suspend,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{Labels: appEnv.Labels("backup")},
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyNever,
							Containers: []corev1.Container{{
								Name:            "backup",
								Image:           appEnv.Spec.App.Image,
								ImagePullPolicy: appEnv.Spec.App.ImagePullPolicy,
								Command:         []string{"/bin/sh", "-c"},
								Args:            []string{"echo backing up to " + appEnv.Spec.Backup.Destination},
							}},
						},
					},
				},
			},
		},
	}
}
