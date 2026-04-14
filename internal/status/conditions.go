// This file centralizes condition manipulation and phase computation. It exists
// so controllers and tests share one status vocabulary and one phase algorithm.
package status

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/meta"

	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
)

func SetCondition(conditions *[]metav1.Condition, conditionType string, status metav1.ConditionStatus, reason, message string, generation int64) {
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: generation,
		LastTransitionTime: metav1.NewTime(time.Now()),
	})
}

func GetCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	return meta.FindStatusCondition(conditions, conditionType)
}

func IsConditionTrue(conditions []metav1.Condition, conditionType string) bool {
	condition := GetCondition(conditions, conditionType)
	return condition != nil && condition.Status == metav1.ConditionTrue
}

func ComputePhase(conditions []metav1.Condition, paused bool, restoring bool) string {
	switch {
	case paused:
		return appsv1beta1.PhasePaused
	case restoring:
		return appsv1beta1.PhaseRestoring
	case IsConditionTrue(conditions, appsv1beta1.ConditionReady):
		return appsv1beta1.PhaseRunning
	case GetCondition(conditions, appsv1beta1.ConditionSpecValid) != nil &&
		GetCondition(conditions, appsv1beta1.ConditionSpecValid).Status == metav1.ConditionFalse:
		return appsv1beta1.PhaseFailed
	case len(conditions) == 0:
		return appsv1beta1.PhasePending
	default:
		return appsv1beta1.PhaseConfiguring
	}
}
