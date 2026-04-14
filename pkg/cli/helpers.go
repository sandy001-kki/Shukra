// This file holds small helper adapters shared by the CLI implementation. It
// exists to keep the command files focused on user-facing behavior.
package cli

import (
	networkingv1 "k8s.io/api/networking/v1"
)

func networkingPathType() networkingv1.PathType {
	return networkingv1.PathTypePrefix
}
