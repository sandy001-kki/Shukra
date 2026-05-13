package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	appsv1alpha1 "github.com/sandy001-kki/Shukra/api/v1alpha1"
	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
	"github.com/sandy001-kki/Shukra/internal/bridge"
	shukrametrics "github.com/sandy001-kki/Shukra/pkg/metrics"
)

func main() {
	var bindAddress string
	var metricsAddress string
	var certFile string
	var keyFile string
	var clientCAFile string
	var requireClientCert bool
	flag.StringVar(&bindAddress, "bind-address", ":50051", "The address where the AIONOS bridge gRPC server listens.")
	flag.StringVar(&metricsAddress, "metrics-bind-address", ":8080", "The address where the bridge metrics endpoint listens.")
	flag.StringVar(&certFile, "tls-cert-file", "/tls/tls.crt", "Bridge server TLS certificate file.")
	flag.StringVar(&keyFile, "tls-key-file", "/tls/tls.key", "Bridge server TLS private key file.")
	flag.StringVar(&clientCAFile, "client-ca-file", "/tls/ca.crt", "Client CA bundle for AIONOS mTLS.")
	flag.BoolVar(&requireClientCert, "require-client-cert", true, "Require and verify AIONOS client certificates.")
	opts := zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	scheme := clientgoscheme.Scheme
	must(appsv1alpha1.AddToScheme(scheme))
	must(appsv1beta1.AddToScheme(scheme))

	kubeClient, err := client.NewWithWatch(ctrl.GetConfigOrDie(), client.Options{Scheme: scheme})
	must(err)

	bridgeServer := bridge.NewServer(kubeClient, scheme)
	ctx := ctrl.SetupSignalHandler()
	must(bridgeServer.EnsureShadowNamespace(ctx))

	shukrametrics.MustRegister()
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(metricsAddress, nil); err != nil {
			ctrl.Log.Error(err, "bridge metrics server stopped")
		}
	}()

	tlsConfig, err := loadTLSConfig(certFile, keyFile, clientCAFile, requireClientCert)
	must(err)
	grpcServer := bridge.NewGRPCServer(tlsConfig, bridgeServer, requireClientCert)

	listener, err := net.Listen("tcp", bindAddress)
	must(err)
	ctrl.Log.Info("starting shukra bridge", "bindAddress", bindAddress, "metricsAddress", metricsAddress)
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			ctrl.Log.Error(err, "bridge gRPC server stopped")
		}
	}()
	<-ctx.Done()
	grpcServer.GracefulStop()
}

func loadTLSConfig(certFile, keyFile, clientCAFile string, requireClientCert bool) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	clientCAs := x509.NewCertPool()
	clientAuth := tls.NoClientCert
	if requireClientCert {
		clientAuth = tls.RequireAndVerifyClientCert
		caBytes, err := os.ReadFile(clientCAFile)
		if err != nil {
			return nil, err
		}
		if ok := clientCAs.AppendCertsFromPEM(caBytes); !ok {
			return nil, fmt.Errorf("client CA bundle contains no certificates")
		}
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		ClientAuth:   clientAuth,
		ClientCAs:    clientCAs,
	}, nil
}

func must(err error) {
	if err != nil {
		ctrl.Log.Error(err, "fatal error")
		os.Exit(1)
	}
}
