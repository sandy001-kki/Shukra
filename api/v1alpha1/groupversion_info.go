// This file registers the served v1alpha1 API group/version. It exists so the
// operator can support upgrades from an older schema through explicit conversion.
// +kubebuilder:object:generate=true
// +groupName=apps.shukra.io
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	GroupVersion  = schema.GroupVersion{Group: "apps.shukra.io", Version: "v1alpha1"}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme
)
