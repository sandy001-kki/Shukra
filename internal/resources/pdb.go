// This file builds the PodDisruptionBudget. It exists so the mutual exclusivity
// contract for minAvailable/maxUnavailable can stay enforced in one place.
package resources

import (
	"fmt"

	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
)

func PDB(appEnv *appsv1beta1.AppEnvironment) (*policyv1.PodDisruptionBudget, error) {
	if appEnv.Spec.Security.PodDisruptionBudget.MinAvailable != nil &&
		appEnv.Spec.Security.PodDisruptionBudget.MaxUnavailable != nil {
		return nil, fmt.Errorf("only one of minAvailable or maxUnavailable may be set")
	}
	return &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Name(appEnv, "pdb"),
			Namespace: appEnv.Namespace,
			Labels:    appEnv.Labels("pdb"),
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			Selector:       &metav1.LabelSelector{MatchLabels: appEnv.Labels("app")},
			MinAvailable:   appEnv.Spec.Security.PodDisruptionBudget.MinAvailable,
			MaxUnavailable: appEnv.Spec.Security.PodDisruptionBudget.MaxUnavailable,
		},
	}, nil
}
