package shadow

import (
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	AnnotationShadow        = "aionos.io/shadow"
	AnnotationShadowSource  = "aionos.io/shadow-source-name"
	AnnotationShadowNS      = "aionos.io/shadow-source-namespace"
	AnnotationShadowPatchID = "aionos.io/patch-id"
	AnnotationShadowTTLSec  = "aionos.io/ttl-seconds"
	AnnotationTrafficWeight = "aionos.io/traffic-weight"
	ShadowNamespace         = "aionos-shadow"
)

func IsShadow(env metav1.Object) bool {
	return env.GetAnnotations()[AnnotationShadow] == "true"
}

func TTLSeconds(env metav1.Object) (int64, bool) {
	val, ok := env.GetAnnotations()[AnnotationShadowTTLSec]
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}
