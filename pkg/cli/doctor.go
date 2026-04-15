// This file implements `shukra doctor`. It exists so users can quickly verify
// whether the local Shukra toolchain, Kubernetes context, CRD, and operator
// control plane are healthy before running lifecycle commands.
package cli

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/discovery"
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
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check whether the local Shukra environment is healthy",
		RunE: func(cmd *cobra.Command, args []string) error {
			results := runDoctorChecks(opts)

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
}

func runDoctorChecks(opts *RootOptions) []doctorResult {
	results := []doctorResult{
		checkBinary("docker", "Docker CLI"),
		checkBinary("kubectl", "kubectl"),
		checkBinary("helm", "Helm"),
	}

	if results[0].Status == "ok" {
		results = append(results, checkDockerEngine())
	} else {
		results = append(results, doctorResult{Name: "Docker engine", Status: "fail", Details: "Skipped because Docker CLI is not available."})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	restConfig, err := buildRESTConfig(opts)
	if err != nil {
		results = append(results,
			doctorResult{Name: "Kube config", Status: "fail", Details: err.Error()},
			doctorResult{Name: "Kubernetes API", Status: "fail", Details: "Skipped because kubeconfig resolution failed."},
			doctorResult{Name: "Shukra CRD", Status: "fail", Details: "Skipped because the Kubernetes API is unavailable."},
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
			doctorResult{Name: "Shukra CRD", Status: "fail", Details: "Skipped because the Kubernetes API is unavailable."},
			doctorResult{Name: "Operator pods", Status: "fail", Details: "Skipped because the Kubernetes API is unavailable."},
			doctorResult{Name: "cert-manager", Status: "fail", Details: "Skipped because the Kubernetes API is unavailable."},
		)
		return results
	}

	serverVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		results = append(results,
			doctorResult{Name: "Kubernetes API", Status: "fail", Details: err.Error()},
			doctorResult{Name: "Shukra CRD", Status: "fail", Details: "Skipped because the Kubernetes API is unavailable."},
			doctorResult{Name: "Operator pods", Status: "fail", Details: "Skipped because the Kubernetes API is unavailable."},
			doctorResult{Name: "cert-manager", Status: "fail", Details: "Skipped because the Kubernetes API is unavailable."},
		)
		return results
	}
	results = append(results, doctorResult{Name: "Kubernetes API", Status: "ok", Details: fmt.Sprintf("Connected to %s.", serverVersion.GitVersion)})

	kubeClient, _, err := buildClient(ctx, opts)
	if err != nil {
		results = append(results,
			doctorResult{Name: "Shukra CRD", Status: "fail", Details: err.Error()},
			doctorResult{Name: "Operator pods", Status: "fail", Details: "Skipped because the controller-runtime client could not be built."},
			doctorResult{Name: "cert-manager", Status: "fail", Details: "Skipped because the controller-runtime client could not be built."},
		)
		return results
	}

	results = append(results, checkShukraCRD(ctx, kubeClient))
	results = append(results, checkOperatorPods(ctx, kubeClient))
	results = append(results, checkCertManager(ctx, kubeClient))
	return results
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

func checkShukraCRD(ctx context.Context, kubeClient ctrlclient.Client) doctorResult {
	list := &appsv1beta1.AppEnvironmentList{}
	if err := kubeClient.List(ctx, list); err != nil {
		return doctorResult{Name: "Shukra CRD", Status: "fail", Details: "AppEnvironment API is not reachable."}
	}
	return doctorResult{Name: "Shukra CRD", Status: "ok", Details: "AppEnvironment API is registered and reachable."}
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
