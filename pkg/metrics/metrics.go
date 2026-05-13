// This file defines exported Prometheus metrics for the operator. It exists in
// pkg so both runtime code and tests can register and inspect the same metrics.
package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	registerOnce sync.Once

	ReconcileDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "shukra_reconcile_duration_seconds",
		Help: "Duration of AppEnvironment reconciliations.",
	}, []string{"namespace", "result"})

	ReconcileFailures = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "shukra_reconcile_failures_total",
		Help: "Count of failed AppEnvironment reconciliations.",
	}, []string{"namespace", "reason"})

	ActiveEnvironments = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "shukra_active_environments",
		Help: "Number of AppEnvironment resources under management.",
	}, []string{"namespace"})

	MigrationsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "shukra_migrations_total",
		Help: "Count of migration executions.",
	}, []string{"namespace", "result"})

	RestoresTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "shukra_restores_total",
		Help: "Count of restore executions.",
	}, []string{"namespace", "result"})

	BackupConfigured = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "shukra_backup_configured",
		Help: "Whether backups are configured for environments.",
	}, []string{"namespace"})

	IntentEvaluationsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "aionos_intent_evaluation_total",
		Help: "Count of AIONOS intent evaluations by result.",
	}, []string{"name", "namespace", "intent_type", "result"})

	ShadowEnvironmentsActive = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "aionos_shadow_environments_active",
		Help: "Number of active AIONOS shadow environments.",
	})

	ShadowEnvironmentTTLExpirationsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "aionos_shadow_environment_ttl_expirations_total",
		Help: "Count of AIONOS shadow environments deleted after TTL expiry.",
	})

	BridgePatchApplicationsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "aionos_bridge_patch_applications_total",
		Help: "Count of AIONOS bridge patch applications.",
	}, []string{"name", "namespace", "applied_by", "result"})

	BridgeStreamConnectionsActive = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "aionos_bridge_stream_connections_active",
		Help: "Active AIONOS bridge streaming RPC connections.",
	}, []string{"stream_type"})
)

func MustRegister() {
	registerOnce.Do(func() {
		crmetrics.Registry.MustRegister(
			ReconcileDuration,
			ReconcileFailures,
			ActiveEnvironments,
			MigrationsTotal,
			RestoresTotal,
			BackupConfigured,
			IntentEvaluationsTotal,
			ShadowEnvironmentsActive,
			ShadowEnvironmentTTLExpirationsTotal,
			BridgePatchApplicationsTotal,
			BridgeStreamConnectionsActive,
		)
	})
}
