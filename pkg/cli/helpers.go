// This file holds small helper adapters shared by the CLI implementation. It
// exists to keep the command files focused on user-facing behavior.
package cli

import (
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func intstrFromInt32(value int32) intstr.IntOrString {
	return intstr.FromInt(int(value))
}

func networkingPathType() networkingv1.PathType {
	return networkingv1.PathTypePrefix
}
