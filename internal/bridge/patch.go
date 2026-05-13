package bridge

import (
	"context"
	"encoding/json"
	"time"

	jsonpatch "github.com/evanphx/json-patch/v5"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	bridgev1 "github.com/sandy001-kki/Shukra/api/bridge/v1"
	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
	shukrametrics "github.com/sandy001-kki/Shukra/pkg/metrics"
)

const (
	AnnotationLastPatchID        = "aionos.io/last-patch-id"
	AnnotationLastPatchReason    = "aionos.io/last-patch-reason"
	AnnotationLastPatchAppliedBy = "aionos.io/last-patch-applied-by"
)

func (s *Server) PatchEnvironment(ctx context.Context, req *bridgev1.PatchRequest) (*bridgev1.PatchResult, error) {
	if err := requiredNameNamespace(req.Name, req.Namespace); err != nil {
		return &bridgev1.PatchResult{Success: false, Error: err.Error()}, nil
	}
	if req.PatchJson == "" {
		return &bridgev1.PatchResult{Success: false, Error: "patch_json is required"}, nil
	}

	key := types.NamespacedName{Name: req.Name, Namespace: req.Namespace}
	var resourceVersion string
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		env := &appsv1beta1.AppEnvironment{}
		if err := s.client.Get(ctx, key, env); err != nil {
			return err
		}
		objectPatch := client.MergeFrom(env.DeepCopy())
		_, statusChanged, err := applyEnvironmentMergePatch(env, []byte(req.PatchJson))
		if err != nil {
			return err
		}
		if env.Annotations == nil {
			env.Annotations = map[string]string{}
		}
		env.Annotations[AnnotationLastPatchID] = req.PatchId
		env.Annotations[AnnotationLastPatchReason] = req.Reason
		env.Annotations[AnnotationLastPatchAppliedBy] = req.AppliedBy
		if err := s.client.Patch(ctx, env, objectPatch); err != nil {
			return err
		}
		if statusChanged {
			fresh := &appsv1beta1.AppEnvironment{}
			if err := s.client.Get(ctx, key, fresh); err != nil {
				return err
			}
			statusPatch := client.MergeFrom(fresh.DeepCopy())
			fresh.Status = env.Status
			if err := s.client.Status().Patch(ctx, fresh, statusPatch); err != nil {
				return err
			}
		}
		s.recordAppEnvironmentEvent(ctx, env, corev1.EventTypeNormal, "AionosPatchApplied", "AIONOS patch "+req.PatchId+" applied by "+req.AppliedBy)
		resourceVersion = env.ResourceVersion
		return s.appendPatchRecord(ctx, key, req, "success")
	})
	result := "success"
	if err != nil {
		result = "error"
		shukrametrics.BridgePatchApplicationsTotal.WithLabelValues(req.Name, req.Namespace, req.AppliedBy, result).Inc()
		return &bridgev1.PatchResult{Success: false, Error: err.Error()}, nil
	}
	shukrametrics.BridgePatchApplicationsTotal.WithLabelValues(req.Name, req.Namespace, req.AppliedBy, result).Inc()
	return &bridgev1.PatchResult{Success: true, ResourceVersion: resourceVersion}, nil
}

func applyEnvironmentMergePatch(env *appsv1beta1.AppEnvironment, patchBytes []byte) (bool, bool, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(patchBytes, &root); err != nil {
		return false, false, err
	}

	specPatch, hasSpec := root["spec"]
	statusPatch, hasStatus := root["status"]
	if !hasSpec && !hasStatus {
		return true, false, applySpecMergePatch(env, patchBytes)
	}

	if hasSpec {
		if err := applySpecMergePatch(env, specPatch); err != nil {
			return false, false, err
		}
	}
	if hasStatus {
		if err := applyStatusMergePatch(env, statusPatch); err != nil {
			return false, false, err
		}
	}
	return hasSpec, hasStatus, nil
}

func applySpecMergePatch(env *appsv1beta1.AppEnvironment, patchBytes []byte) error {
	current, err := json.Marshal(env.Spec)
	if err != nil {
		return err
	}
	merged, err := jsonpatch.MergePatch(current, patchBytes)
	if err != nil {
		return err
	}
	var spec appsv1beta1.AppEnvironmentSpec
	if err := json.Unmarshal(merged, &spec); err != nil {
		return err
	}
	env.Spec = spec
	return nil
}

func applyStatusMergePatch(env *appsv1beta1.AppEnvironment, patchBytes []byte) error {
	current, err := json.Marshal(env.Status)
	if err != nil {
		return err
	}
	merged, err := jsonpatch.MergePatch(current, patchBytes)
	if err != nil {
		return err
	}
	var status appsv1beta1.AppEnvironmentStatus
	if err := json.Unmarshal(merged, &status); err != nil {
		return err
	}
	env.Status = status
	return nil
}

func (s *Server) appendPatchRecord(ctx context.Context, key types.NamespacedName, req *bridgev1.PatchRequest, result string) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		env := &appsv1beta1.AppEnvironment{}
		if err := s.client.Get(ctx, key, env); err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return err
		}
		patch := client.MergeFrom(env.DeepCopy())
		env.Status.PatchHistory = append([]appsv1beta1.PatchRecord{{
			PatchID:   req.PatchId,
			AppliedBy: req.AppliedBy,
			Reason:    req.Reason,
			AppliedAt: metav1.NewTime(time.Now()),
			Result:    result,
		}}, env.Status.PatchHistory...)
		if len(env.Status.PatchHistory) > 20 {
			env.Status.PatchHistory = env.Status.PatchHistory[:20]
		}
		return s.client.Status().Patch(ctx, env, patch)
	})
}
