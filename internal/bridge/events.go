package bridge

import (
	"context"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	bridgev1 "github.com/sandy001-kki/Shukra/api/bridge/v1"
	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
	shukrametrics "github.com/sandy001-kki/Shukra/pkg/metrics"
)

func (s *Server) StreamReconcileEvents(req *bridgev1.ReconcileEventRequest, stream bridgev1.AionosBridge_StreamReconcileEventsServer) error {
	shukrametrics.BridgeStreamConnectionsActive.WithLabelValues("reconcile_events").Inc()
	defer shukrametrics.BridgeStreamConnectionsActive.WithLabelValues("reconcile_events").Dec()

	seen := map[string]struct{}{}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		if err := s.sendReconcileEvents(req, stream, seen); err != nil {
			return err
		}
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-ticker.C:
		}
	}
}

func (s *Server) sendReconcileEvents(req *bridgev1.ReconcileEventRequest, stream bridgev1.AionosBridge_StreamReconcileEventsServer, seen map[string]struct{}) error {
	list := &corev1.EventList{}
	opts := []client.ListOption{}
	if req.Namespace != "" {
		opts = append(opts, client.InNamespace(req.Namespace))
	}
	if err := s.client.List(stream.Context(), list, opts...); err != nil {
		return err
	}
	for _, event := range list.Items {
		if event.InvolvedObject.Kind != "AppEnvironment" {
			continue
		}
		if req.Name != "" && event.InvolvedObject.Name != req.Name {
			continue
		}
		key := string(event.UID) + "/" + event.ResourceVersion
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if err := stream.Send(&bridgev1.ReconcileEvent{
			Name:          event.InvolvedObject.Name,
			Namespace:     event.Namespace,
			EventType:     reconcileEventType(event.Reason),
			ResourceKind:  event.InvolvedObject.Kind,
			Message:       event.Message,
			Error:         eventError(event),
			TimestampUnix: event.LastTimestamp.Unix(),
		}); err != nil {
			return err
		}
	}
	return nil
}

func reconcileEventType(reason string) string {
	switch {
	case strings.Contains(reason, "Failure"), strings.Contains(reason, "Failed"):
		return "ReconcileFailure"
	case strings.Contains(reason, "Migration"):
		return "MigrationRun"
	case strings.Contains(reason, "Restore"):
		return "RestoreRun"
	case strings.Contains(reason, "Intent"):
		return "IntentViolation"
	case strings.Contains(reason, "Patch"):
		return "PatchApplied"
	default:
		return "ReconcileSuccess"
	}
}

func eventError(event corev1.Event) string {
	if event.Type == corev1.EventTypeWarning {
		return event.Message
	}
	return ""
}

func (s *Server) recordAppEnvironmentEvent(ctx context.Context, env *appsv1beta1.AppEnvironment, eventType, reason, message string) {
	now := metav1.Now()
	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: env.Name + "-",
			Namespace:    env.Namespace,
		},
		InvolvedObject: corev1.ObjectReference{
			APIVersion: appsv1beta1.GroupVersion.String(),
			Kind:       "AppEnvironment",
			Namespace:  env.Namespace,
			Name:       env.Name,
			UID:        env.UID,
		},
		Type:           eventType,
		Reason:         reason,
		Message:        message,
		FirstTimestamp: now,
		LastTimestamp:  now,
		Count:          1,
		Source:         corev1.EventSource{Component: "shukra-bridge"},
	}
	_ = s.client.Create(ctx, event)
}
