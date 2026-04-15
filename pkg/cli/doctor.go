// This file implements `shukra doctor`. It exists so users can quickly verify
// whether the local Shukra toolchain, Kubernetes context, CRD, and operator
// control plane are healthy before running lifecycle commands.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
	"github.com/spf13/cobra"
)

type doctorResult struct {
	Name    string
	Status  string
	Details string
}

func newDoctorCommand(opts *RootOptions) *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check whether the local Shukra environment is healthy",
		RunE: func(cmd *cobra.Command, args []string) error {
			results := runDoctorChecks(opts)
			if output == "json" {
				payload, err := json.MarshalIndent(results, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(payload))
				if hasDoctorFailure(results) {
					return fmt.Errorf("one or more doctor checks failed")
				}
				return nil
			}

			printTitle(cmd.OutOrStdout(), "Shukra Doctor")
			for _, result := range results {
				fmt.Fprintf(cmd.OutOrStdout(), "%-22s %-5s %s\n", result.Name, doctorBadge(result.Status), result.Details)
			}

			if hasDoctorFailure(results) {
				return fmt.Errorf("one or more doctor checks failed")
			}
			fmt.Fprintln(cmd.OutOrStdout(), success("Environment looks healthy."))
			return nil
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "summary", "Output format: summary or json.")
	return cmd
}

func runDoctorChecks(opts *RootOptions) []doctorResult {
	rawConfig, rawConfigErr := loadRawConfig(opts)
	results := []doctorResult{
		checkBinary("kubectl", "kubectl"),
		checkBinary("helm", "Helm"),
	}

	if rawConfigErr == nil {
		results = append(results, checkContext(rawConfig, opts))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	restConfig, err := buildRESTConfig(opts)
	if err != nil {
		results = append(results,
			doctorResult{Name: "Kube config", Status: "fail", Details: err.Error()},
			doctorResult{Name: "Kubernetes API", Status: "fail", Details: "Skipped because kubeconfig resolution failed."},
			doctorResult{Name: "Docker CLI", Status: "warn", Details: "Docker is only required for local image builds and kind bootstrap."},
			doctorResult{Name: "Docker engine", Status: "warn", Details: "Docker engine check skipped because Kubernetes connection is unavailable."},
			doctorResult{Name: "Shukra CRD", Status: "fail", Details: "Skipped because the Kubernetes API is unavailable."},
			doctorResult{Name: "Cluster nodes", Status: "fail", Details: "Skipped because the Kubernetes API is unavailable."},
			doctorResult{Name: "Operator pods", Status: "fail", Details: "Skipped because the Kubernetes API is unavailable."},
			doctorResult{Name: "cert-manager", Status: "fail", Details: "Skipped because the Kubernetes API is unavailable."},
		)
		return results
	}

	results = append(results, doctorResult{Name: "Kube config", Status: "ok", Details: "Kubeconfig loaded successfully."})

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		results = append(results,
			doctorResult{Name: "Kubernetes API", Status: "fail", Details: err.Error()},
			doctorResult{Name: "Docker CLI", Status: "warn", Details: "Docker is only required for local image builds and kind bootstrap."},
			doctorResult{Name: "Docker engine", Status: "warn", Details: "Docker engine check skipped because Kubernetes API discovery failed."},
			doctorResult{Name: "Shukra CRD", Status: "fail", Details: "Skipped because the Kubernetes API is unavailable."},
			doctorResult{Name: "Cluster nodes", Status: "fail", Details: "Skipped because the Kubernetes API is unavailable."},
			doctorResult{Name: "Operator pods", Status: "fail", Details: "Skipped because the Kubernetes API is unavailable."},
			doctorResult{Name: "cert-manager", Status: "fail", Details: "Skipped because the Kubernetes API is unavailable."},
		)
		return results
	}

	serverVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		results = append(results,
			doctorResult{Name: "Kubernetes API", Status: "fail", Details: err.Error()},
			doctorResult{Name: "Docker CLI", Status: "warn", Details: "Docker is only required for local image builds and kind bootstrap."},
			doctorResult{Name: "Docker engine", Status: "warn", Details: "Docker engine check skipped because Kubernetes API discovery failed."},
			doctorResult{Name: "Shukra CRD", Status: "fail", Details: "Skipped because the Kubernetes API is unavailable."},
			doctorResult{Name: "Cluster nodes", Status: "fail", Details: "Skipped because the Kubernetes API is unavailable."},
			doctorResult{Name: "Operator pods", Status: "fail", Details: "Skipped because the Kubernetes API is unavailable."},
			doctorResult{Name: "cert-manager", Status: "fail", Details: "Skipped because the Kubernetes API is unavailable."},
		)
		return results
	}
	results = append(results, doctorResult{Name: "Kubernetes API", Status: "ok", Details: fmt.Sprintf("Connected to %s.", serverVersion.GitVersion)})
	results = append(results, checkDockerSupport(rawConfig, rawConfigErr))

	kubeClient, _, err := buildClient(ctx, opts)
	if err != nil {
		results = append(results,
			doctorResult{Name: "Shukra CRD", Status: "fail", Details: err.Error()},
			doctorResult{Name: "Cluster nodes", Status: "fail", Details: "Skipped because the controller-runtime client could not be built."},
			doctorResult{Name: "Operator pods", Status: "fail", Details: "Skipped because the controller-runtime client could not be built."},
			doctorResult{Name: "cert-manager", Status: "fail", Details: "Skipped because the controller-runtime client could not be built."},
		)
		return results
	}

	results = append(results, checkShukraCRD(ctx, kubeClient))
	results = append(results, checkClusterNodes(ctx, kubeClient))
	results = append(results, checkOperatorPods(ctx, kubeClient))
	results = append(results, checkCertManager(ctx, kubeClient))
	return results
}

func loadRawConfig(opts *RootOptions) (*clientcmdapi.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = opts.Kubeconfig
	return loadingRules.Load()
}

func checkBinary(binary, name string) doctorResult {
	if _, err := exec.LookPath(binary); err != nil {
		return doctorResult{Name: name, Status: "fail", Details: fmt.Sprintf("%s is not available on PATH.", binary)}
	}
	return doctorResult{Name: name, Status: "ok", Details: fmt.Sprintf("%s is available.", binary)}
}

func checkDockerEngine() doctorResult {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "version", "--format", "{{.Server.Version}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return doctorResult{Name: "Docker engine", Status: "fail", Details: "Docker engine is not responding."}
	}
	version := string(output)
	version = trimWhitespace(version)
	if version == "" {
		version = "version unavailable"
	}
	return doctorResult{Name: "Docker engine", Status: "ok", Details: fmt.Sprintf("Docker engine is running (%s).", version)}
}

func checkContext(rawConfig *clientcmdapi.Config, opts *RootOptions) doctorResult {
	contextName := opts.Context
	if contextName == "" {
		contextName = rawConfig.CurrentContext
	}
	if contextName == "" {
		return doctorResult{Name: "Kube context", Status: "warn", Details: "No current context is set in kubeconfig."}
	}
	contextConfig, ok := rawConfig.Contexts[contextName]
	if !ok {
		return doctorResult{Name: "Kube context", Status: "warn", Details: fmt.Sprintf("Context %s was not found in kubeconfig.", contextName)}
	}
	clusterName := contextConfig.Cluster
	if clusterName == "" {
		clusterName = "unknown cluster"
	}
	return doctorResult{Name: "Kube context", Status: "ok", Details: fmt.Sprintf("Using context %s on %s.", contextName, clusterName)}
}

func checkDockerSupport(rawConfig *clientcmdapi.Config, rawErr error) doctorResult {
	if rawErr != nil {
		return doctorResult{Name: "Docker support", Status: "warn", Details: "Docker check skipped because kubeconfig context could not be inspected."}
	}
	contextName := rawConfig.CurrentContext
	contextConfig, ok := rawConfig.Contexts[contextName]
	clusterName := ""
	if ok {
		clusterName = contextConfig.Cluster
	}
	localCluster := strings.Contains(strings.ToLower(contextName), "kind") ||
		strings.Contains(strings.ToLower(contextName), "minikube") ||
		strings.Contains(strings.ToLower(contextName), "k3d") ||
		strings.Contains(strings.ToLower(clusterName), "kind") ||
		strings.Contains(strings.ToLower(clusterName), "minikube") ||
		strings.Contains(strings.ToLower(clusterName), "k3d")

	_, lookErr := exec.LookPath("docker")
	if localCluster {
		if lookErr != nil {
			return doctorResult{Name: "Docker support", Status: "warn", Details: "Local cluster context detected. Docker is usually required for local image build and kind-style workflows."}
		}
		engine := checkDockerEngine()
		if engine.Status == "ok" {
			return doctorResult{Name: "Docker support", Status: "ok", Details: "Docker is ready for local build and bootstrap workflows."}
		}
		return doctorResult{Name: "Docker support", Status: "warn", Details: "Local cluster context detected, but Docker engine is not currently reachable."}
	}

	if lookErr != nil {
		return doctorResult{Name: "Docker support", Status: "ok", Details: "Docker is not required for bring-your-own-cluster installs."}
	}
	engine := checkDockerEngine()
	if engine.Status == "ok" {
		return doctorResult{Name: "Docker support", Status: "ok", Details: "Docker is available for optional local image builds."}
	}
	return doctorResult{Name: "Docker support", Status: "warn", Details: "Docker is optional for this cluster, but the local engine is not currently reachable."}
}

func checkShukraCRD(ctx context.Context, kubeClient ctrlclient.Client) doctorResult {
	list := &appsv1beta1.AppEnvironmentList{}
	if err := kubeClient.List(ctx, list); err != nil {
		return doctorResult{Name: "Shukra CRD", Status: "fail", Details: "AppEnvironment API is not reachable."}
	}
	return doctorResult{Name: "Shukra CRD", Status: "ok", Details: "AppEnvironment API is registered and reachable."}
}

func checkClusterNodes(ctx context.Context, kubeClient ctrlclient.Client) doctorResult {
	nodeList := &corev1.NodeList{}
	if err := kubeClient.List(ctx, nodeList); err != nil {
		return doctorResult{Name: "Cluster nodes", Status: "fail", Details: "Unable to list cluster nodes."}
	}
	if len(nodeList.Items) == 0 {
		return doctorResult{Name: "Cluster nodes", Status: "warn", Details: "Connected successfully, but no nodes were listed."}
	}
	profile := detectClusterProfile(nodeList.Items)
	return doctorResult{Name: "Cluster nodes", Status: "ok", Details: fmt.Sprintf("%d node(s) detected. Profile: %s.", len(nodeList.Items), profile)}
}

func checkOperatorPods(ctx context.Context, kubeClient ctrlclient.Client) doctorResult {
	podList := &corev1.PodList{}
	if err := kubeClient.List(ctx, podList); err != nil {
		return doctorResult{Name: "Operator pods", Status: "fail", Details: "Unable to list Pods."}
	}

	running := 0
	for _, pod := range podList.Items {
		if pod.Namespace == "shukra-system" && trimWhitespace(pod.Labels["app.kubernetes.io/name"]) == "shukra-operator" && isPodReady(pod) {
			running++
		}
	}
	if running == 0 {
		return doctorResult{Name: "Operator pods", Status: "fail", Details: "No ready shukra-operator pod found in shukra-system."}
	}
	return doctorResult{Name: "Operator pods", Status: "ok", Details: fmt.Sprintf("%d ready shukra-operator pod(s) found.", running)}
}

func checkCertManager(ctx context.Context, kubeClient ctrlclient.Client) doctorResult {
	podList := &corev1.PodList{}
	if err := kubeClient.List(ctx, podList); err != nil {
		return doctorResult{Name: "cert-manager", Status: "fail", Details: "Unable to list Pods."}
	}

	ready := 0
	for _, pod := range podList.Items {
		if pod.Namespace == "cert-manager" && isPodReady(pod) {
			ready++
		}
	}
	if ready == 0 {
		return doctorResult{Name: "cert-manager", Status: "warn", Details: "No ready cert-manager pods found."}
	}
	return doctorResult{Name: "cert-manager", Status: "ok", Details: fmt.Sprintf("%d ready cert-manager pod(s) found.", ready)}
}

func doctorBadge(status string) string {
	switch status {
	case "ok":
		return success("OK")
	case "warn":
		return muted("WARN")
	default:
		return "FAIL"
	}
}

func hasDoctorFailure(results []doctorResult) bool {
	for _, result := range results {
		if result.Status == "fail" {
			return true
		}
	}
	return false
}

func isPodReady(pod corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodRunning {
		return false
	}
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func trimWhitespace(value string) string {
	return strings.TrimSpace(value)
}

func detectClusterProfile(nodes []corev1.Node) string {
	for _, node := range nodes {
		name := strings.ToLower(node.Name)
		switch {
		case strings.Contains(name, "kind"):
			return "kind"
		case strings.Contains(name, "minikube"):
			return "minikube"
		case strings.Contains(name, "k3d"):
			return "k3d"
		}
		for key, value := range node.Labels {
			label := strings.ToLower(key + "=" + value)
			switch {
			case strings.Contains(label, "eks.amazonaws.com"):
				return "eks"
			case strings.Contains(label, "cloud.google.com/gke"):
				return "gke"
			case strings.Contains(label, "kubernetes.azure.com"):
				return "aks"
			case strings.Contains(label, "k3s.io"):
				return "k3s"
			}
		}
	}
	return "generic kubernetes"
}
