// This file registers the canonical v1beta1 API group/version. It exists so
// controller-runtime and the Kubernetes API machinery can encode, decode, and
// store AppEnvironment objects using the storage version.
// +kubebuilder:object:generate=true
// +groupName=apps.shukra.io
package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	GroupVersion  = schema.GroupVersion{Group: "apps.shukra.io", Version: "v1beta1"}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme
)
