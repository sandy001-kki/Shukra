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
		)
	})
}
