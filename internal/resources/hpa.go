// This file builds the HorizontalPodAutoscaler. It exists as a pure builder so
// autoscaling rules can evolve without complicating reconcile orchestration.
package resources

import (
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
)

func HPA(appEnv *appsv1beta1.AppEnvironment) *autoscalingv2.HorizontalPodAutoscaler {
	metrics := make([]autoscalingv2.MetricSpec, 0, 2)
	if appEnv.Spec.Autoscaling.TargetCPUUtilizationPercentage != nil {
		metrics = append(metrics, autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: "cpu",
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: appEnv.Spec.Autoscaling.TargetCPUUtilizationPercentage,
				},
			},
		})
	}
	if appEnv.Spec.Autoscaling.TargetMemoryUtilizationPercentage != nil {
		metrics = append(metrics, autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: "memory",
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: appEnv.Spec.Autoscaling.TargetMemoryUtilizationPercentage,
				},
			},
		})
	}
	return &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Name(appEnv, "hpa"),
			Namespace: appEnv.Namespace,
			Labels:    appEnv.Labels("hpa"),
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       Name(appEnv, "deployment"),
			},
			MinReplicas: appEnv.Spec.Autoscaling.MinReplicas,
			MaxReplicas: appEnv.Spec.Autoscaling.MaxReplicas,
			Metrics:     metrics,
		},
	}
}
