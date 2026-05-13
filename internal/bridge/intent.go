package bridge

import (
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	bridgev1 "github.com/sandy001-kki/Shukra/api/bridge/v1"
	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
	shukrametrics "github.com/sandy001-kki/Shukra/pkg/metrics"
)

func (s *Server) StreamIntentViolations(req *bridgev1.IntentViolationRequest, stream bridgev1.AionosBridge_StreamIntentViolationsServer) error {
	shukrametrics.BridgeStreamConnectionsActive.WithLabelValues("intent_violations").Inc()
	defer shukrametrics.BridgeStreamConnectionsActive.WithLabelValues("intent_violations").Dec()

	seen := map[string]struct{}{}
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		if err := s.sendIntentViolations(req, stream, seen); err != nil {
			return err
		}
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-ticker.C:
		}
	}
}

func (s *Server) sendIntentViolations(req *bridgev1.IntentViolationRequest, stream bridgev1.AionosBridge_StreamIntentViolationsServer, seen map[string]struct{}) error {
	list := &appsv1beta1.AppEnvironmentList{}
	opts := []client.ListOption{}
	if req.Namespace != "" {
		opts = append(opts, client.InNamespace(req.Namespace))
	}
	if err := s.client.List(stream.Context(), list, opts...); err != nil {
		return err
	}
	for _, env := range list.Items {
		if req.Name != "" && env.Name != req.Name {
			continue
		}
		for _, condition := range env.Status.IntentHealth {
			if condition.Status != "Violated" {
				continue
			}
			key := env.Namespace + "/" + env.Name + "/" + condition.Type + "/" + condition.LastCheck.String()
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			if err := stream.Send(&bridgev1.IntentViolationEvent{
				Name:          env.Name,
				Namespace:     env.Namespace,
				IntentType:    condition.Type,
				Declared:      condition.Declared,
				Measured:      condition.Measured,
				Severity:      severityForIntent(condition.Type),
				TimestampUnix: condition.LastCheck.Unix(),
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func severityForIntent(intentType string) string {
	if intentType == "RequireNetworkPolicy" {
		return "Critical"
	}
	return "Warning"
}
