// This file boots the controller manager, webhooks, health probes, metrics, and
// leader election settings. It exists as the single runtime entrypoint for the
// operator Deployment and for `make run` during local development.
package main

import (
	"flag"
	"os"

	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	appsv1alpha1 "github.com/sandy001-kki/Shukra/api/v1alpha1"
	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
	"github.com/sandy001-kki/Shukra/controllers"
	"github.com/sandy001-kki/Shukra/pkg/metrics"
	"github.com/sandy001-kki/Shukra/webhooks"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(appsv1alpha1.AddToScheme(scheme))
	utilruntime.Must(appsv1beta1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var probeAddr string
	var webhookPort int
	var watchNamespace string
	var maxConcurrentReconciles int

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8443", "The address where the metrics endpoint serves Prometheus metrics.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address used for health and readiness probes.")
	flag.IntVar(&webhookPort, "webhook-port", 9443, "The HTTPS port used by admission and conversion webhooks.")
	flag.StringVar(&watchNamespace, "watch-namespace", "", "Namespace to watch. Empty means all namespaces.")
	flag.IntVar(&maxConcurrentReconciles, "max-concurrent-reconciles", 5, "Maximum concurrent reconciliation workers.")

	opts := zap.Options{
		Development: false,
		Level:       zapcore.InfoLevel,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	leaderElectionNamespace := os.Getenv("POD_NAMESPACE")
	if leaderElectionNamespace == "" {
		leaderElectionNamespace = "shukra-system"
	}

	managerOptions := ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		WebhookServer: webhook.NewServer(webhook.Options{
			Port: webhookPort,
		}),
		// Leader election is required in production so only one replica actively
		// mutates shared resources while standby replicas remain hot for failover.
		LeaderElection:          true,
		LeaderElectionID:        "shukra-operator-leader",
		LeaderElectionNamespace: leaderElectionNamespace,
	}
	if watchNamespace != "" {
		managerOptions.Cache = cache.Options{DefaultNamespaces: map[string]cache.Config{watchNamespace: {}}}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), managerOptions)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	metrics.MustRegister()

	eventRecorder := mgr.GetEventRecorderFor("shukra-operator")
	if err := (&controllers.AppEnvironmentReconciler{
		Client:                   mgr.GetClient(),
		Scheme:                   mgr.GetScheme(),
		EventRecorder:            eventRecorder,
		MaxConcurrentReconciles:  maxConcurrentReconciles,
		WatchNamespace:           watchNamespace,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AppEnvironment")
		os.Exit(1)
	}

	if err := webhooks.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to register webhooks")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up readiness check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
