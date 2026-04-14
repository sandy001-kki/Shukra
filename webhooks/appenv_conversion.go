// This file registers the conversion webhook endpoint through controller-runtime.
// It exists so Kubernetes can convert served v1alpha1 resources to the v1beta1
// hub storage version and back during read/write operations.
package webhooks

import (
	ctrl "sigs.k8s.io/controller-runtime"
)

func SetupWithManager(mgr ctrl.Manager) error {
	if err := setupDefaulter(mgr); err != nil {
		return err
	}
	if err := setupValidator(mgr); err != nil {
		return err
	}
	return nil
}
