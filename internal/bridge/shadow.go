package bridge

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	bridgev1 "github.com/sandy001-kki/Shukra/api/bridge/v1"
	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
	"github.com/sandy001-kki/Shukra/internal/shadow"
)

var dnsLabelInvalid = regexp.MustCompile(`[^a-z0-9-]+`)

func (s *Server) CreateShadowEnvironment(ctx context.Context, req *bridgev1.ShadowCreateRequest) (*bridgev1.ShadowResult, error) {
	if err := requiredNameNamespace(req.SourceName, req.SourceNamespace); err != nil {
		return &bridgev1.ShadowResult{Success: false, Error: err.Error()}, nil
	}
	if err := s.EnsureShadowNamespace(ctx); err != nil {
		return &bridgev1.ShadowResult{Success: false, Error: err.Error()}, nil
	}

	source := &appsv1beta1.AppEnvironment{}
	if err := s.client.Get(ctx, types.NamespacedName{Name: req.SourceName, Namespace: req.SourceNamespace}, source); err != nil {
		return &bridgev1.ShadowResult{Success: false, Error: err.Error()}, nil
	}

	shadowEnv := source.DeepCopy()
	shadowEnv.TypeMeta = source.TypeMeta
	shadowEnv.ObjectMeta = metav1.ObjectMeta{
		Name:      shadowName(req.SourceName, req.PatchId),
		Namespace: shadow.ShadowNamespace,
		Labels: map[string]string{
			"aionos.io/shadow":   "true",
			"aionos.io/patch-id": req.PatchId,
		},
		Annotations: map[string]string{
			shadow.AnnotationShadow:        "true",
			shadow.AnnotationShadowSource:  req.SourceName,
			shadow.AnnotationShadowNS:      req.SourceNamespace,
			shadow.AnnotationShadowPatchID: req.PatchId,
			shadow.AnnotationTrafficWeight: "0",
		},
	}
	if req.TtlSeconds > 0 {
		shadowEnv.Annotations[shadow.AnnotationShadowTTLSec] = fmt.Sprintf("%d", req.TtlSeconds)
	}
	if req.ImageOverride != "" {
		shadowEnv.Spec.App.Image = req.ImageOverride
	}
	if req.SpecPatchJson != "" {
		if err := applySpecMergePatch(shadowEnv, []byte(req.SpecPatchJson)); err != nil {
			return &bridgev1.ShadowResult{Success: false, Error: err.Error()}, nil
		}
	}
	shadowEnv.Status = appsv1beta1.AppEnvironmentStatus{}

	if err := s.client.Create(ctx, shadowEnv); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return &bridgev1.ShadowResult{Success: true, ShadowName: shadowEnv.Name, ShadowNamespace: shadowEnv.Namespace}, nil
		}
		return &bridgev1.ShadowResult{Success: false, Error: err.Error()}, nil
	}
	s.recordAppEnvironmentEvent(ctx, shadowEnv, corev1.EventTypeNormal, "AionosShadowCreated", "AIONOS shadow environment created")
	return &bridgev1.ShadowResult{Success: true, ShadowName: shadowEnv.Name, ShadowNamespace: shadowEnv.Namespace}, nil
}

func (s *Server) DeleteShadowEnvironment(ctx context.Context, req *bridgev1.ShadowDeleteRequest) (*bridgev1.DeleteResult, error) {
	namespace := req.ShadowNamespace
	if namespace == "" {
		namespace = shadow.ShadowNamespace
	}
	if req.ShadowName == "" {
		return &bridgev1.DeleteResult{Success: false, Error: "shadow_name is required"}, nil
	}
	env := &appsv1beta1.AppEnvironment{}
	if err := s.client.Get(ctx, types.NamespacedName{Name: req.ShadowName, Namespace: namespace}, env); err != nil {
		if apierrors.IsNotFound(err) {
			return &bridgev1.DeleteResult{Success: true}, nil
		}
		return &bridgev1.DeleteResult{Success: false, Error: err.Error()}, nil
	}
	if !shadow.IsShadow(env) {
		return &bridgev1.DeleteResult{Success: false, Error: "refusing to delete non-shadow environment"}, nil
	}
	if err := s.client.Delete(ctx, env); err != nil && !apierrors.IsNotFound(err) {
		return &bridgev1.DeleteResult{Success: false, Error: err.Error()}, nil
	}
	return &bridgev1.DeleteResult{Success: true}, nil
}

func shadowName(sourceName, patchID string) string {
	base := strings.ToLower(sourceName + "-" + patchID + "-shadow")
	base = dnsLabelInvalid.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if len(base) > 63 {
		base = base[:63]
		base = strings.TrimRight(base, "-")
	}
	if base == "" {
		return "aionos-shadow"
	}
	return base
}
