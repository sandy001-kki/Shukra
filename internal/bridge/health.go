package bridge

import (
	"context"
	"encoding/json"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	bridgev1 "github.com/sandy001-kki/Shukra/api/bridge/v1"
	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
	"github.com/sandy001-kki/Shukra/internal/resources"
	shukrametrics "github.com/sandy001-kki/Shukra/pkg/metrics"
)

func (s *Server) StreamEnvironmentHealth(req *bridgev1.HealthStreamRequest, stream bridgev1.AionosBridge_StreamEnvironmentHealthServer) error {
	shukrametrics.BridgeStreamConnectionsActive.WithLabelValues("health").Inc()
	defer shukrametrics.BridgeStreamConnectionsActive.WithLabelValues("health").Dec()

	ticker := time.NewTicker(intervalSeconds(req.IntervalSeconds))
	defer ticker.Stop()
	opts := []client.ListOption{}
	if req.Namespace != "" {
		opts = append(opts, client.InNamespace(req.Namespace))
	}
	var watcher watch.Interface
	defer func() {
		if watcher != nil {
			watcher.Stop()
		}
	}()

	for {
		if watcher == nil {
			nextWatcher, err := s.client.Watch(stream.Context(), &appsv1beta1.AppEnvironmentList{}, opts...)
			if err != nil {
				return err
			}
			watcher = nextWatcher
		}
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case event, ok := <-watcher.ResultChan():
			if !ok {
				watcher.Stop()
				watcher = nil
				continue
			}
			if event.Type == watch.Deleted {
				continue
			}
			env, ok := event.Object.(*appsv1beta1.AppEnvironment)
			if !ok || (req.Name != "" && env.Name != req.Name) {
				continue
			}
			if err := s.sendHealthEvent(stream.Context(), env, stream); err != nil {
				return err
			}
		case <-ticker.C:
			if err := s.sendHealthSnapshot(stream.Context(), req, stream); err != nil {
				return err
			}
		}
	}
}

func (s *Server) ListEnvironments(ctx context.Context, req *bridgev1.ListRequest) (*bridgev1.EnvironmentList, error) {
	list := &appsv1beta1.AppEnvironmentList{}
	opts := []client.ListOption{}
	if req.Namespace != "" {
		opts = append(opts, client.InNamespace(req.Namespace))
	}
	if err := s.client.List(ctx, list, opts...); err != nil {
		return nil, err
	}
	out := &bridgev1.EnvironmentList{Environments: make([]*bridgev1.EnvironmentSummary, 0, len(list.Items))}
	for _, env := range list.Items {
		summary := &bridgev1.EnvironmentSummary{
			Name:         env.Name,
			Namespace:    env.Namespace,
			Phase:        env.Status.Phase,
			FailureCount: env.Status.FailureCount,
			LastError:    env.Status.LastError,
		}
		if env.Status.LastSuccessfulReconcileTime != nil {
			summary.LastSuccessfulReconcileUnix = env.Status.LastSuccessfulReconcileTime.Unix()
		}
		out.Environments = append(out.Environments, summary)
	}
	return out, nil
}

func (s *Server) GetEnvironment(ctx context.Context, req *bridgev1.GetRequest) (*bridgev1.EnvironmentDetail, error) {
	if err := requiredNameNamespace(req.Name, req.Namespace); err != nil {
		return nil, err
	}
	env := &appsv1beta1.AppEnvironment{}
	if err := s.client.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, env); err != nil {
		return nil, err
	}
	specJSON, _ := json.Marshal(env.Spec)
	statusJSON, _ := json.Marshal(env.Status)
	return &bridgev1.EnvironmentDetail{
		Name:           env.Name,
		Namespace:      env.Namespace,
		Phase:          env.Status.Phase,
		SpecJson:       string(specJSON),
		StatusJson:     string(statusJSON),
		Conditions:     conditionProtos(env.Status.Conditions),
		IntentHealth:   intentConditionProtos(env.Status.IntentHealth),
		ChildResources: childResourceNames(env.Status.ChildResources),
	}, nil
}

func (s *Server) sendHealthSnapshot(ctx context.Context, req *bridgev1.HealthStreamRequest, stream bridgev1.AionosBridge_StreamEnvironmentHealthServer) error {
	list := &appsv1beta1.AppEnvironmentList{}
	opts := []client.ListOption{}
	if req.Namespace != "" {
		opts = append(opts, client.InNamespace(req.Namespace))
	}
	if err := s.client.List(ctx, list, opts...); err != nil {
		return err
	}
	for i := range list.Items {
		env := &list.Items[i]
		if req.Name != "" && env.Name != req.Name {
			continue
		}
		if err := s.sendHealthEvent(ctx, env, stream); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) sendHealthEvent(ctx context.Context, env *appsv1beta1.AppEnvironment, stream bridgev1.AionosBridge_StreamEnvironmentHealthServer) error {
	ready, desired := s.deploymentReplicas(ctx, env)
	return stream.Send(&bridgev1.HealthEvent{
		Name:            env.Name,
		Namespace:       env.Namespace,
		Phase:           env.Status.Phase,
		ReadyReplicas:   ready,
		DesiredReplicas: desired,
		FailureCount:    env.Status.FailureCount,
		LastError:       env.Status.LastError,
		Conditions:      conditionProtos(env.Status.Conditions),
		IntentHealth:    intentConditionProtos(env.Status.IntentHealth),
		TimestampUnix:   time.Now().Unix(),
	})
}

func (s *Server) deploymentReplicas(ctx context.Context, env *appsv1beta1.AppEnvironment) (int32, int32) {
	deployment := &appsv1.Deployment{}
	name := env.Status.ChildResources.DeploymentName
	if name == "" {
		name = resources.Name(env, "deployment")
	}
	if err := s.client.Get(ctx, types.NamespacedName{Name: name, Namespace: env.Namespace}, deployment); err != nil {
		return 0, env.EffectiveReplicas()
	}
	desired := env.EffectiveReplicas()
	if deployment.Spec.Replicas != nil {
		desired = *deployment.Spec.Replicas
	}
	return deployment.Status.ReadyReplicas, desired
}

func conditionProtos(conditions []metav1.Condition) []*bridgev1.ConditionProto {
	out := make([]*bridgev1.ConditionProto, 0, len(conditions))
	for _, condition := range conditions {
		out = append(out, &bridgev1.ConditionProto{
			Type:               condition.Type,
			Status:             string(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
			LastTransitionUnix: condition.LastTransitionTime.Unix(),
		})
	}
	return out
}

func intentConditionProtos(conditions []appsv1beta1.IntentCondition) []*bridgev1.IntentConditionProto {
	out := make([]*bridgev1.IntentConditionProto, 0, len(conditions))
	for _, condition := range conditions {
		out = append(out, &bridgev1.IntentConditionProto{
			Type:          condition.Type,
			Status:        condition.Status,
			Measured:      condition.Measured,
			Declared:      condition.Declared,
			Message:       condition.Message,
			LastCheckUnix: condition.LastCheck.Unix(),
		})
	}
	return out
}

func childResourceNames(resources appsv1beta1.ChildResources) []string {
	values := []string{
		resources.DeploymentName,
		resources.ServiceName,
		resources.ConfigMapName,
		resources.HPAName,
		resources.IngressName,
		resources.MigrationJobName,
		resources.RestoreJobName,
		resources.BackupCronJobName,
		resources.NetworkPolicyName,
		resources.PDBName,
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
