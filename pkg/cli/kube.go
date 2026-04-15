// This file centralizes Kubernetes API client construction for the CLI. It
// exists so every command uses the same scheme and kubeconfig resolution rules.
package cli

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	appsv1alpha1 "github.com/sandy001-kki/Shukra/api/v1alpha1"
	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
)

func buildRESTConfig(opts *RootOptions) (*rest.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = opts.Kubeconfig
	configOverrides := &clientcmd.ConfigOverrides{}
	if opts.Context != "" {
		configOverrides.CurrentContext = opts.Context
	}

	restConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("build kube config: %w", err)
	}
	return restConfig, nil
}

func buildClient(ctx context.Context, opts *RootOptions) (ctrlclient.Client, *runtime.Scheme, error) {
	_ = ctx

	restConfig, err := buildRESTConfig(opts)
	if err != nil {
		return nil, nil, err
	}

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(appsv1alpha1.AddToScheme(scheme))
	utilruntime.Must(appsv1beta1.AddToScheme(scheme))

	kubeClient, err := ctrlclient.New(restConfig, ctrlclient.Options{Scheme: scheme})
	if err != nil {
		return nil, nil, fmt.Errorf("build controller-runtime client: %w", err)
	}

	return kubeClient, scheme, nil
}
