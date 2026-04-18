// This file implements the Shukra Web Console. It exists to give users a
// useful browser dashboard for cluster health, AppEnvironment status, and
// troubleshooting without requiring a separate frontend build pipeline.
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
	"github.com/spf13/cobra"
)

type consoleCondition struct {
	Type    string
	Status  string
	Reason  string
	Message string
}

type consoleEnvironment struct {
	AnchorID     string
	Name         string
	Namespace    string
	Phase        string
	URL          string
	Ready        string
	FailureCount int32
	LastError    string
	LastSuccess  string
	Resources    []string
	Conditions   []consoleCondition
}

type consoleOperatorPod struct {
	Name   string
	Status string
	Node   string
}

type consolePage struct {
	GeneratedAt       string
	Cluster           string
	Namespace         string
	Count             int
	RunningCount      int
	DegradedCount     int
	FailedCount       int
	PausedCount       int
	ReadyCount        int
	OperatorNamespace string
	LocalhostAddress  string
	OperatorPods      []consoleOperatorPod
	CommandProfiles   []consoleCommandProfile
	Items             []consoleEnvironment
}

type consoleActionPage struct {
	GeneratedAt string
	Title       string
	Command     string
	Output      string
	Success     bool
}

type consoleCommandProfile struct {
	Key         string
	Label       string
	Description string
	NeedsTarget bool
}

func newConsoleCommand(opts *RootOptions) *cobra.Command {
	var addr string

	cmd := &cobra.Command{
		Use:   "console",
		Short: "Run a local Shukra Web Console",
		RunE: func(cmd *cobra.Command, args []string) error {
			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
				defer cancel()

				page, err := buildConsolePage(ctx, opts, addr)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if err := consoleTemplate.Execute(w, page); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			})
			mux.HandleFunc("/api/environments", func(w http.ResponseWriter, r *http.Request) {
				ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
				defer cancel()

				page, err := buildConsolePage(ctx, opts, addr)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(page)
			})
			mux.HandleFunc("/action", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					http.Error(w, "POST required", http.StatusMethodNotAllowed)
					return
				}
				if err := r.ParseForm(); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}

				ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
				defer cancel()

				result := executeConsoleAction(
					ctx,
					opts,
					r.FormValue("action"),
					r.FormValue("namespace"),
					r.FormValue("name"),
				)
				statusCode := http.StatusOK
				if !result.Success {
					statusCode = http.StatusBadRequest
				}
				w.WriteHeader(statusCode)
				if err := consoleActionTemplate.Execute(w, result); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			})

			fmt.Fprintf(cmd.OutOrStdout(), "Shukra Web Console running at http://%s\n", addr)
			return http.ListenAndServe(addr, mux)
		},
	}

	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:8088", "Address to bind the local console server.")
	return cmd
}

func consoleCommandProfiles() []consoleCommandProfile {
	return []consoleCommandProfile{
		{Key: "doctor", Label: "Run Doctor", Description: "Check Docker, kubectl, helm, CRDs, operator Pods, and cert-manager."},
		{Key: "diagnose-operator", Label: "Diagnose Operator", Description: "Show operator Pod health and placement."},
		{Key: "operator-logs", Label: "Tail Operator Logs", Description: "Fetch the latest controller logs."},
		{Key: "apply-basic", Label: "Apply Basic Example", Description: "Apply examples/basic.yaml to the current cluster."},
		{Key: "cluster-appenvs", Label: "List AppEnvironments", Description: "Run kubectl get appenvironments across namespaces."},
		{Key: "cluster-nodes", Label: "List Cluster Nodes", Description: "Inspect Kubernetes nodes and scheduling state."},
		{Key: "namespace-pods", Label: "List Namespace Pods", Description: "Show pods in the selected namespace."},
		{Key: "namespace-services", Label: "List Services and ConfigMaps", Description: "Show service and config objects in the selected namespace."},
		{Key: "namespace-jobs", Label: "List Jobs and CronJobs", Description: "Show batch resources in the selected namespace."},
		{Key: "env-summary", Label: "Environment Summary", Description: "Show phase, readiness, failures, and conditions for one AppEnvironment.", NeedsTarget: true},
		{Key: "env-yaml", Label: "Get Environment YAML", Description: "Print the AppEnvironment resource as YAML.", NeedsTarget: true},
		{Key: "env-describe", Label: "Describe Environment", Description: "Run kubectl describe against one AppEnvironment.", NeedsTarget: true},
		{Key: "env-resources", Label: "List Managed Resources", Description: "Show deploy, svc, cm, ingress, HPA, jobs, CronJobs, network policies, and PDBs for the namespace.", NeedsTarget: true},
		{Key: "diagnose-env", Label: "Diagnose Environment", Description: "Run the focused Shukra diagnosis view for one environment.", NeedsTarget: true},
		{Key: "pause-env", Label: "Pause Environment", Description: "Set spec.paused=true on one AppEnvironment.", NeedsTarget: true},
		{Key: "resume-env", Label: "Resume Environment", Description: "Set spec.paused=false on one AppEnvironment.", NeedsTarget: true},
		{Key: "delete-env", Label: "Delete Environment", Description: "Delete one AppEnvironment and let finalizers clean up.", NeedsTarget: true},
	}
}

func buildConsolePage(ctx context.Context, opts *RootOptions, addr string) (*consolePage, error) {
	kubeClient, _, err := buildClient(ctx, opts)
	if err != nil {
		return nil, err
	}

	var list appsv1beta1.AppEnvironmentList
	if err := kubeClient.List(ctx, &list); err != nil {
		return nil, fmt.Errorf("list AppEnvironments: %w", err)
	}

	items := make([]consoleEnvironment, 0, len(list.Items))
	page := &consolePage{
		GeneratedAt:       time.Now().Format(time.RFC1123),
		Cluster:           currentContext(opts),
		Namespace:         opts.Namespace,
		OperatorNamespace: "shukra-system",
		LocalhostAddress:  addr,
		CommandProfiles:   consoleCommandProfiles(),
	}

	for _, item := range list.Items {
		env := consoleEnvironment{
			AnchorID:     consoleAnchorID(item.Namespace, item.Name),
			Name:         item.Name,
			Namespace:    item.Namespace,
			Phase:        emptyDash(item.Status.Phase),
			URL:          emptyDash(item.Status.URL),
			Ready:        consoleConditionStatus(item.Status.Conditions, appsv1beta1.ConditionReady),
			FailureCount: item.Status.FailureCount,
			LastError:    emptyDash(item.Status.LastError),
			LastSuccess:  formatConsoleTime(item.Status.LastSuccessfulReconcileTime),
			Resources: sortResources([]string{
				item.Status.ChildResources.ConfigMapName,
				item.Status.ChildResources.ServiceName,
				item.Status.ChildResources.DeploymentName,
				item.Status.ChildResources.HPAName,
				item.Status.ChildResources.IngressName,
				item.Status.ChildResources.MigrationJobName,
				item.Status.ChildResources.RestoreJobName,
				item.Status.ChildResources.BackupCronJobName,
				item.Status.ChildResources.NetworkPolicyName,
				item.Status.ChildResources.PDBName,
			}),
			Conditions: make([]consoleCondition, 0, len(item.Status.Conditions)),
		}

		for _, condition := range item.Status.Conditions {
			env.Conditions = append(env.Conditions, consoleCondition{
				Type:    condition.Type,
				Status:  string(condition.Status),
				Reason:  condition.Reason,
				Message: condition.Message,
			})
		}

		switch env.Phase {
		case appsv1beta1.PhaseRunning:
			page.RunningCount++
		case appsv1beta1.PhasePaused:
			page.PausedCount++
		case appsv1beta1.PhaseFailed:
			page.FailedCount++
		case appsv1beta1.PhaseDegraded:
			page.DegradedCount++
		}
		if env.Ready == "True" {
			page.ReadyCount++
		}

		items = append(items, env)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Namespace == items[j].Namespace {
			return items[i].Name < items[j].Name
		}
		return items[i].Namespace < items[j].Namespace
	})

	page.Items = items
	page.Count = len(items)
	page.OperatorPods = listOperatorPods(ctx, kubeClient, page.OperatorNamespace)
	return page, nil
}

func executeConsoleAction(ctx context.Context, opts *RootOptions, actionType, namespace, name string) *consoleActionPage {
	result := &consoleActionPage{
		GeneratedAt: time.Now().Format(time.RFC1123),
		Title:       "Shukra Console Action",
		Success:     false,
	}

	switch actionType {
	case "doctor":
		results := runDoctorChecks(opts)
		var builder strings.Builder
		for _, item := range results {
			fmt.Fprintf(&builder, "%-22s %-5s %s\n", item.Name, strings.ToUpper(item.Status), item.Details)
		}
		result.Title = "Doctor"
		result.Command = "shukra doctor"
		result.Output = builder.String()
		result.Success = !hasDoctorFailure(results)
	case "diagnose-operator":
		result.Title = "Diagnose Operator"
		result.Command = "shukra diagnose operator"
		output, err := diagnoseOperatorOutput(ctx, opts)
		result.Output = output
		result.Success = err == nil
	case "operator-logs":
		result.Title = "Operator Logs"
		result.Command = "kubectl logs -n shukra-system deploy/shukra-operator --tail=120"
		output, err := runKubectlCapture(ctx, opts, "logs", "-n", "shukra-system", "deploy/shukra-operator", "--tail=120")
		result.Output = output
		result.Success = err == nil
	case "apply-basic":
		result.Title = "Apply Basic Example"
		result.Command = "kubectl apply -f examples/basic.yaml"
		output, err := runKubectlCapture(ctx, opts, "apply", "-f", "examples/basic.yaml")
		result.Output = output
		result.Success = err == nil
	case "cluster-appenvs":
		result.Title = "List AppEnvironments"
		result.Command = "kubectl get appenvironments.apps.shukra.io -A -o wide"
		output, err := runKubectlCapture(ctx, opts, "get", "appenvironments.apps.shukra.io", "-A", "-o", "wide")
		result.Output = output
		result.Success = err == nil
	case "cluster-nodes":
		result.Title = "List Cluster Nodes"
		result.Command = "kubectl get nodes -o wide"
		output, err := runKubectlCapture(ctx, opts, "get", "nodes", "-o", "wide")
		result.Output = output
		result.Success = err == nil
	case "namespace-pods":
		ns := consoleNamespaceOrDefault(namespace, opts)
		result.Title = fmt.Sprintf("List Pods in %s", ns)
		result.Command = fmt.Sprintf("kubectl get pods -n %s -o wide", ns)
		output, err := runKubectlCapture(ctx, opts, "get", "pods", "-n", ns, "-o", "wide")
		result.Output = output
		result.Success = err == nil
	case "namespace-services":
		ns := consoleNamespaceOrDefault(namespace, opts)
		result.Title = fmt.Sprintf("List Services and ConfigMaps in %s", ns)
		result.Command = fmt.Sprintf("kubectl get svc,cm -n %s", ns)
		output, err := runKubectlCapture(ctx, opts, "get", "svc,cm", "-n", ns)
		result.Output = output
		result.Success = err == nil
	case "namespace-jobs":
		ns := consoleNamespaceOrDefault(namespace, opts)
		result.Title = fmt.Sprintf("List Jobs and CronJobs in %s", ns)
		result.Command = fmt.Sprintf("kubectl get jobs,cronjobs -n %s", ns)
		output, err := runKubectlCapture(ctx, opts, "get", "jobs,cronjobs", "-n", ns)
		result.Output = output
		result.Success = err == nil
	case "env-summary":
		ns := consoleNamespaceOrDefault(namespace, opts)
		result.Title = fmt.Sprintf("Environment Summary %s/%s", ns, name)
		result.Command = fmt.Sprintf("shukra diagnose env %s -n %s", name, ns)
		output, err := diagnoseEnvironmentOutput(ctx, opts, ns, name)
		result.Output = output
		result.Success = err == nil
	case "env-yaml":
		ns := consoleNamespaceOrDefault(namespace, opts)
		result.Title = fmt.Sprintf("Environment YAML %s/%s", ns, name)
		result.Command = fmt.Sprintf("kubectl get appenvironment %s -n %s -o yaml", name, ns)
		output, err := runKubectlCapture(ctx, opts, "get", "appenvironment", name, "-n", ns, "-o", "yaml")
		result.Output = output
		result.Success = err == nil
	case "env-describe":
		ns := consoleNamespaceOrDefault(namespace, opts)
		result.Title = fmt.Sprintf("Describe Environment %s/%s", ns, name)
		result.Command = fmt.Sprintf("kubectl describe appenvironment %s -n %s", name, ns)
		output, err := runKubectlCapture(ctx, opts, "describe", "appenvironment", name, "-n", ns)
		result.Output = output
		result.Success = err == nil
	case "env-resources":
		ns := consoleNamespaceOrDefault(namespace, opts)
		result.Title = fmt.Sprintf("Managed Resources in %s for %s", ns, name)
		result.Command = fmt.Sprintf("kubectl get deploy,svc,cm,ingress,hpa,jobs,cronjobs,networkpolicies,poddisruptionbudgets -n %s", ns)
		output, err := runKubectlCapture(ctx, opts, "get", "deploy,svc,cm,ingress,hpa,jobs,cronjobs,networkpolicies,poddisruptionbudgets", "-n", ns)
		result.Output = output
		result.Success = err == nil
	case "diagnose-env":
		ns := consoleNamespaceOrDefault(namespace, opts)
		result.Title = fmt.Sprintf("Diagnose %s/%s", ns, name)
		result.Command = fmt.Sprintf("shukra diagnose env %s -n %s", name, ns)
		output, err := diagnoseEnvironmentOutput(ctx, opts, ns, name)
		result.Output = output
		result.Success = err == nil
	case "pause-env":
		ns := consoleNamespaceOrDefault(namespace, opts)
		result.Title = fmt.Sprintf("Pause %s/%s", ns, name)
		result.Command = fmt.Sprintf("shukra env pause %s -n %s", name, ns)
		output, err := mutateEnvironmentPause(ctx, opts, ns, name, true)
		result.Output = output
		result.Success = err == nil
	case "resume-env":
		ns := consoleNamespaceOrDefault(namespace, opts)
		result.Title = fmt.Sprintf("Resume %s/%s", ns, name)
		result.Command = fmt.Sprintf("shukra env resume %s -n %s", name, ns)
		output, err := mutateEnvironmentPause(ctx, opts, ns, name, false)
		result.Output = output
		result.Success = err == nil
	case "delete-env":
		ns := consoleNamespaceOrDefault(namespace, opts)
		result.Title = fmt.Sprintf("Delete %s/%s", ns, name)
		result.Command = fmt.Sprintf("shukra env delete %s -n %s", name, ns)
		output, err := deleteEnvironment(ctx, opts, ns, name)
		result.Output = output
		result.Success = err == nil
	default:
		result.Output = fmt.Sprintf("Unknown action: %s", actionType)
	}

	if !result.Success && strings.TrimSpace(result.Output) == "" {
		result.Output = "The requested action failed without additional output."
	}
	return result
}

func diagnoseOperatorOutput(ctx context.Context, opts *RootOptions) (string, error) {
	kubeClient, _, err := buildClient(ctx, opts)
	if err != nil {
		return err.Error(), err
	}
	pods := listOperatorPods(ctx, kubeClient, "shukra-system")
	var builder strings.Builder
	builder.WriteString("Shukra operator state\n\n")
	for _, pod := range pods {
		fmt.Fprintf(&builder, "- %s phase=%s node=%s\n", pod.Name, pod.Status, pod.Node)
	}
	return builder.String(), nil
}

func diagnoseEnvironmentOutput(ctx context.Context, opts *RootOptions, namespace, name string) (string, error) {
	kubeClient, _, err := buildClient(ctx, opts)
	if err != nil {
		return err.Error(), err
	}

	appEnv := &appsv1beta1.AppEnvironment{}
	key := types.NamespacedName{Name: name, Namespace: namespace}
	if err := kubeClient.Get(ctx, key, appEnv); err != nil {
		return fmt.Sprintf("Unable to load %s/%s: %v", namespace, name, err), err
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "AppEnvironment %s/%s\n\n", namespace, name)
	fmt.Fprintf(&builder, "Phase: %s\n", emptyDash(appEnv.Status.Phase))
	fmt.Fprintf(&builder, "Ready: %s\n", consoleConditionStatus(appEnv.Status.Conditions, appsv1beta1.ConditionReady))
	fmt.Fprintf(&builder, "Paused: %t\n", appEnv.Spec.Paused)
	fmt.Fprintf(&builder, "Failure count: %d\n", appEnv.Status.FailureCount)
	fmt.Fprintf(&builder, "URL: %s\n", emptyDash(appEnv.Status.URL))
	fmt.Fprintf(&builder, "Last error: %s\n", emptyDash(appEnv.Status.LastError))
	builder.WriteString("\nConditions:\n")
	for _, condition := range appEnv.Status.Conditions {
		fmt.Fprintf(&builder, "- %s=%s reason=%s message=%q\n", condition.Type, condition.Status, condition.Reason, condition.Message)
	}
	return builder.String(), nil
}

func mutateEnvironmentPause(ctx context.Context, opts *RootOptions, namespace, name string, paused bool) (string, error) {
	kubeClient, _, err := buildClient(ctx, opts)
	if err != nil {
		return err.Error(), err
	}

	appEnv := &appsv1beta1.AppEnvironment{}
	key := types.NamespacedName{Name: name, Namespace: namespace}
	if err := kubeClient.Get(ctx, key, appEnv); err != nil {
		return fmt.Sprintf("Unable to load %s/%s: %v", namespace, name, err), err
	}

	appEnv.Spec.Paused = paused
	if err := kubeClient.Update(ctx, appEnv); err != nil {
		return fmt.Sprintf("Unable to update %s/%s: %v", namespace, name, err), err
	}

	state := "resumed"
	if paused {
		state = "paused"
	}
	return fmt.Sprintf("AppEnvironment %s/%s %s successfully.", namespace, name, state), nil
}

func deleteEnvironment(ctx context.Context, opts *RootOptions, namespace, name string) (string, error) {
	kubeClient, _, err := buildClient(ctx, opts)
	if err != nil {
		return err.Error(), err
	}

	appEnv := &appsv1beta1.AppEnvironment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	if err := kubeClient.Delete(ctx, appEnv); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Sprintf("Unable to delete %s/%s: %v", namespace, name, err), err
	}
	return fmt.Sprintf("Delete requested for AppEnvironment %s/%s. Finalizer cleanup, if needed, will finish before removal.", namespace, name), nil
}

func runKubectlCapture(ctx context.Context, opts *RootOptions, args ...string) (string, error) {
	fullArgs := appendKubectlConnectionArgs(opts, args)
	cmd := exec.CommandContext(ctx, "kubectl", fullArgs...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	output := strings.TrimSpace(stdout.String())
	errOutput := strings.TrimSpace(stderr.String())
	switch {
	case output != "" && errOutput != "":
		output = output + "\n\n" + errOutput
	case output == "":
		output = errOutput
	}
	if err != nil {
		if strings.TrimSpace(output) == "" {
			output = err.Error()
		}
		return output, err
	}
	if strings.TrimSpace(output) == "" {
		output = "Command completed successfully."
	}
	return output, nil
}

func listOperatorPods(ctx context.Context, kubeClient client.Client, namespace string) []consoleOperatorPod {
	var podList corev1.PodList
	if err := kubeClient.List(ctx, &podList, client.InNamespace(namespace)); err != nil {
		return []consoleOperatorPod{{Name: "unavailable", Status: err.Error(), Node: "-"}}
	}

	pods := make([]consoleOperatorPod, 0, len(podList.Items))
	for _, pod := range podList.Items {
		if !strings.Contains(pod.Name, "shukra-operator") {
			continue
		}
		pods = append(pods, consoleOperatorPod{
			Name:   pod.Name,
			Status: string(pod.Status.Phase),
			Node:   pod.Spec.NodeName,
		})
	}
	if len(pods) == 0 {
		return []consoleOperatorPod{{Name: "none", Status: "NotFound", Node: "-"}}
	}
	sort.Slice(pods, func(i, j int) bool { return pods[i].Name < pods[j].Name })
	return pods
}

func currentContext(opts *RootOptions) string {
	if opts.Context != "" {
		return opts.Context
	}
	return "current kube context"
}

func sortResources(resources []string) []string {
	filtered := make([]string, 0, len(resources))
	for _, resource := range resources {
		if strings.TrimSpace(resource) != "" {
			filtered = append(filtered, resource)
		}
	}
	sort.Strings(filtered)
	if len(filtered) == 0 {
		return []string{"-"}
	}
	return filtered
}

func consoleConditionStatus(conditions []metav1.Condition, conditionType string) string {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return string(condition.Status)
		}
	}
	return "Unknown"
}

func formatConsoleTime(value *metav1.Time) string {
	if value == nil || value.IsZero() {
		return "-"
	}
	return value.Time.Format("2006-01-02 15:04:05 MST")
}

func consoleNamespaceOrDefault(namespace string, opts *RootOptions) string {
	namespace = strings.TrimSpace(namespace)
	if namespace != "" {
		return namespace
	}
	if strings.TrimSpace(opts.Namespace) != "" {
		return opts.Namespace
	}
	return "default"
}

func consoleAnchorID(namespace, name string) string {
	replacer := strings.NewReplacer("/", "-", ".", "-", "_", "-", " ", "-")
	return "env-" + replacer.Replace(namespace) + "-" + replacer.Replace(name)
}

var consoleTemplate = template.Must(template.New("console").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta http-equiv="refresh" content="15">
  <title>Shukra Web Console</title>
  <style>
    :root {
      color-scheme: light;
      --page: #f6f1e8;
      --surface: #fffdf9;
      --surface-2: #f1e7d8;
      --line: #1f2a1f;
      --ink: #1f2a1f;
      --muted: #5c6756;
      --sage: #7f9378;
      --sage-deep: #425342;
      --amber: #b7791f;
      --amber-soft: #ead6b6;
      --danger: #9b3d2f;
      --success: #2f6a4f;
      --shadow: rgba(31, 42, 31, 0.08);
    }
    * { box-sizing: border-box; }
    html { scroll-behavior: smooth; }
    body {
      margin: 0;
      background: var(--page);
      color: var(--ink);
      font-family: "Segoe UI", Arial, sans-serif;
    }
    .wrap {
      max-width: 1480px;
      margin: 0 auto;
      padding: 24px;
    }
    .hero, .toolbar, .panel, .card, .table-wrap, .jump-list, .shell {
      background: var(--surface);
      border: 1px solid var(--line);
      box-shadow: 0 14px 30px var(--shadow);
    }
    .hero, .toolbar, .panel, .card, .jump-list, .shell { padding: 22px; }
    h1, h2, h3 {
      margin: 0;
      font-weight: 700;
      letter-spacing: -0.02em;
    }
    h1 {
      font-size: 44px;
      line-height: 1.02;
      max-width: 11ch;
    }
    h2 { font-size: 23px; margin-bottom: 12px; }
    h3 { font-size: 16px; margin-bottom: 10px; }
    p { line-height: 1.6; }
    .eyebrow {
      display: inline-block;
      border: 1px solid var(--sage-deep);
      color: var(--sage-deep);
      background: var(--surface-2);
      padding: 6px 10px;
      font-size: 11px;
      text-transform: uppercase;
      letter-spacing: .12em;
      margin-bottom: 12px;
    }
    .hero-grid {
      display: grid;
      grid-template-columns: 1.45fr .95fr;
      gap: 20px;
      align-items: start;
    }
    .sub {
      margin: 14px 0 0;
      max-width: 920px;
      font-size: 16px;
      color: var(--muted);
    }
    .token-row, .actions, .mini-actions {
      display: flex;
      gap: 10px;
      flex-wrap: wrap;
    }
    .token {
      display: inline-block;
      border: 1px solid var(--sage);
      background: var(--surface-2);
      color: var(--sage-deep);
      padding: 6px 10px;
      font-size: 12px;
      text-transform: uppercase;
      letter-spacing: .08em;
    }
    .stats {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
      gap: 12px;
      margin-top: 18px;
    }
    .stat {
      border: 1px solid var(--sage);
      background: var(--surface);
      padding: 14px;
    }
    .label {
      font-size: 12px;
      text-transform: uppercase;
      letter-spacing: .08em;
      color: var(--muted);
    }
    .value {
      font-size: 30px;
      margin-top: 8px;
      color: var(--ink);
    }
    .shell {
      background: var(--ink);
      color: #f3ecdf;
      border-color: var(--ink);
    }
    .shell h3 { color: #f8f2e7; }
    .code, pre {
      margin: 0;
      white-space: pre-wrap;
      word-break: break-word;
      font-family: Consolas, monospace;
      font-size: 13px;
      line-height: 1.55;
    }
    .toolbar {
      margin-top: 18px;
      display: grid;
      gap: 16px;
    }
    .section-head {
      display: flex;
      justify-content: space-between;
      gap: 12px;
      align-items: end;
    }
    .section-head p {
      margin: 0;
      max-width: 720px;
      font-size: 13px;
      color: var(--muted);
    }
    .toolbar-grid {
      display: grid;
      grid-template-columns: 1.4fr 1fr;
      gap: 12px;
    }
    .search-wrap label {
      display: block;
      font-size: 12px;
      text-transform: uppercase;
      letter-spacing: .08em;
      color: var(--muted);
      margin-bottom: 8px;
    }
    .search-wrap input, .search-wrap select {
      width: 100%;
      border: 1px solid var(--sage);
      background: #fff;
      color: var(--ink);
      padding: 12px 14px;
      font-size: 14px;
      outline: none;
    }
    .search-wrap input:focus, .search-wrap select:focus {
      border-color: var(--amber);
      box-shadow: 0 0 0 3px rgba(183, 121, 31, 0.15);
    }
    .workspace {
      display: grid;
      grid-template-columns: 320px minmax(0, 1fr);
      gap: 18px;
      margin-top: 18px;
      align-items: start;
    }
    .sidebar {
      display: grid;
      gap: 18px;
      position: sticky;
      top: 18px;
    }
    .footnote {
      font-size: 12px;
      color: var(--muted);
      margin-top: 0;
    }
    .jump-links {
      display: flex;
      flex-direction: column;
      gap: 8px;
    }
    .jump-links a, button, a.button {
      display: inline-block;
      text-decoration: none;
      cursor: pointer;
      border: 1px solid var(--line);
      background: var(--surface);
      color: var(--ink);
      padding: 10px 14px;
      font-size: 12px;
      text-transform: uppercase;
      letter-spacing: .08em;
      transition: background-color .18s ease, color .18s ease, border-color .18s ease, transform .18s ease;
    }
    .jump-links a:hover, .jump-links a:focus-visible, button:hover, a.button:hover, button:focus-visible, a.button:focus-visible {
      background: var(--ink);
      color: #fff;
      border-color: var(--ink);
      outline: none;
      transform: translateY(-1px);
    }
    .table-wrap {
      overflow: auto;
      background: var(--surface);
    }
    table {
      width: 100%;
      border-collapse: collapse;
    }
    th, td {
      border-bottom: 1px solid rgba(31, 42, 31, 0.16);
      padding: 14px;
      vertical-align: top;
      text-align: left;
    }
    th {
      font-size: 12px;
      text-transform: uppercase;
      letter-spacing: .08em;
      color: var(--muted);
      background: rgba(127, 147, 120, 0.08);
    }
    .phase {
      display: inline-block;
      border: 1px solid var(--sage);
      background: var(--surface-2);
      color: var(--sage-deep);
      padding: 6px 10px;
      font-size: 12px;
      text-transform: uppercase;
      letter-spacing: .08em;
    }
    .row-link {
      color: var(--ink);
      text-decoration: none;
      border-bottom: 1px solid var(--amber);
    }
    .row-link:hover, .row-link:focus-visible {
      color: var(--amber);
      outline: none;
    }
    .cards {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(430px, 1fr));
      gap: 18px;
      margin-top: 18px;
    }
    .card {
      background: var(--surface);
    }
    .card-head {
      display: flex;
      justify-content: space-between;
      gap: 12px;
      align-items: start;
    }
    .card-name {
      font-size: 25px;
      font-weight: 700;
    }
    .card-ns {
      color: var(--muted);
      margin-top: 4px;
      font-size: 14px;
    }
    .meta {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 10px;
      margin: 16px 0;
    }
    .meta-box {
      border: 1px solid rgba(31, 42, 31, 0.2);
      background: rgba(127, 147, 120, 0.08);
      padding: 12px;
      min-height: 72px;
    }
    .meta-box strong {
      display: block;
      font-size: 11px;
      text-transform: uppercase;
      letter-spacing: .08em;
      margin-bottom: 6px;
      color: var(--muted);
    }
    .list {
      margin: 0;
      padding-left: 18px;
    }
    .list li { margin: 4px 0; }
    .conditions {
      display: grid;
      gap: 10px;
      margin-top: 12px;
    }
    .condition {
      border: 1px solid rgba(31, 42, 31, 0.18);
      background: rgba(255, 255, 255, 0.72);
      padding: 11px;
    }
    .condition .type {
      font-size: 12px;
      text-transform: uppercase;
      color: var(--sage-deep);
    }
    .condition .reason {
      font-size: 12px;
      margin-top: 6px;
      color: var(--muted);
    }
    .condition .message {
      font-size: 13px;
      margin-top: 6px;
    }
    .operator-list li {
      margin: 8px 0;
      padding-bottom: 8px;
      border-bottom: 1px dashed rgba(31, 42, 31, 0.24);
    }
    .footer {
      font-size: 13px;
      margin-top: 18px;
      color: var(--muted);
    }
    @media (max-width: 980px) {
      .hero-grid, .toolbar-grid, .workspace { grid-template-columns: 1fr; }
      .sidebar { position: static; }
      .cards { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <div class="wrap">
    <section class="hero">
      <div class="hero-grid">
        <div>
          <div class="eyebrow">Shukra Operations Surface</div>
          <h1>Shukra Web Console</h1>
          <p class="sub">A local operator workspace for AppEnvironment health, operator visibility, and safe action flows. This console stays on <strong>{{.LocalhostAddress}}</strong> so it can use your kube context without exposing a remote control plane.</p>
          <div class="token-row" style="margin-top:14px;">
            <span class="token">Cluster {{.Cluster}}</span>
            <span class="token">Namespace {{.Namespace}}</span>
            <span class="token">Operator {{.OperatorNamespace}}</span>
          </div>
          <div class="stats">
            <div class="stat"><div class="label">Visible environments</div><div class="value">{{.Count}}</div></div>
            <div class="stat"><div class="label">Ready</div><div class="value">{{.ReadyCount}}</div></div>
            <div class="stat"><div class="label">Running</div><div class="value">{{.RunningCount}}</div></div>
            <div class="stat"><div class="label">Paused</div><div class="value">{{.PausedCount}}</div></div>
            <div class="stat"><div class="label">Failed</div><div class="value">{{.FailedCount}}</div></div>
            <div class="stat"><div class="label">Degraded</div><div class="value">{{.DegradedCount}}</div></div>
          </div>
        </div>
        <div class="shell">
          <h3>Console Contract</h3>
          <pre class="code">Generated: {{.GeneratedAt}}
Address: {{.LocalhostAddress}}
JSON API: /api/environments

Browser actions are terminal-backed.
Commands stay whitelisted and local.
This UI is for trusted operator work.</pre>
        </div>
      </div>
    </section>

    <section class="toolbar" aria-label="Workspace toolbar">
      <div class="section-head">
        <div>
          <h2>Workspace</h2>
          <p>Use filters to narrow the visible environments, jump directly to a resource card, or run safe local operations from the action rail.</p>
        </div>
        <div class="mini-actions">
          <a class="button" href="/">Refresh</a>
          <a class="button" href="/api/environments">Open JSON API</a>
        </div>
      </div>
      <div class="toolbar-grid">
        <div class="search-wrap">
          <label for="env-filter">Search environments</label>
          <input id="env-filter" type="search" placeholder="Search by name, namespace, or phase" autocomplete="off">
        </div>
        <div class="search-wrap">
          <label for="phase-filter">Filter by phase</label>
          <select id="phase-filter">
            <option value="">All phases</option>
            <option value="running">Running</option>
            <option value="paused">Paused</option>
            <option value="failed">Failed</option>
            <option value="degraded">Degraded</option>
            <option value="configuring">Configuring</option>
            <option value="restoring">Restoring</option>
            <option value="deleting">Deleting</option>
          </select>
        </div>
      </div>
    </section>

    <section class="workspace">
      <aside class="sidebar">
        <section class="panel">
          <h2>Operator Actions</h2>
          <p class="footnote">These actions are explicit and local. They execute whitelisted Shukra and kubectl workflows and return the command output in the browser.</p>
          <div class="actions">
            <form method="post" action="/action"><input type="hidden" name="action" value="doctor"><button type="submit">Run Doctor</button></form>
            <form method="post" action="/action"><input type="hidden" name="action" value="diagnose-operator"><button type="submit">Diagnose Operator</button></form>
            <form method="post" action="/action"><input type="hidden" name="action" value="operator-logs"><button type="submit">Tail Operator Logs</button></form>
            <form method="post" action="/action"><input type="hidden" name="action" value="apply-basic"><button type="submit">Apply Basic Example</button></form>
          </div>
        </section>

        <section class="panel">
          <h2>Operator Pods</h2>
          <ul class="list operator-list">
            {{range .OperatorPods}}
            <li><strong>{{.Name}}</strong><br>phase={{.Status}}<br>node={{.Node}}</li>
            {{end}}
          </ul>
        </section>

        <section class="jump-list">
          <h3>Jump to Environment</h3>
          <div class="jump-links">
            {{range .Items}}
            <a href="#{{.AnchorID}}">{{.Namespace}} / {{.Name}}</a>
            {{end}}
          </div>
        </section>
      </aside>

      <div>
        <section class="panel" style="margin-bottom:18px;">
          <div class="section-head">
            <div>
              <h2>Command Center</h2>
              <p>Run safe terminal-backed operations from the browser. Pick a command profile, optionally target an environment, and Shukra will execute the matching local workflow.</p>
            </div>
          </div>
          <form method="post" action="/action">
            <div class="toolbar-grid">
              <div class="search-wrap">
                <label for="command-profile">Command profile</label>
                <select id="command-profile" name="action">
                  {{range .CommandProfiles}}
                  <option value="{{.Key}}" data-needs-target="{{if .NeedsTarget}}true{{else}}false{{end}}">{{.Label}} — {{.Description}}</option>
                  {{end}}
                </select>
              </div>
              <div class="search-wrap">
                <label for="command-namespace">Namespace</label>
                <input id="command-namespace" name="namespace" type="text" placeholder="default" value="{{if .Namespace}}{{.Namespace}}{{else}}default{{end}}">
              </div>
            </div>
            <div class="toolbar-grid" style="margin-top:12px;">
              <div class="search-wrap">
                <label for="command-name">Environment name</label>
                <input id="command-name" name="name" type="text" placeholder="basic-app">
              </div>
              <div class="search-wrap" style="display:flex; align-items:end;">
                <button type="submit" style="width:100%;">Run Command</button>
              </div>
            </div>
            <p class="footnote" id="command-help" style="margin-top:12px;">Tip: choose an environment-aware command such as Environment Summary, Diagnose Environment, Pause, Resume, or Delete when you want browser-driven control over a specific AppEnvironment.</p>
          </form>
        </section>

        <section class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Environment</th>
                <th>Phase</th>
                <th>Ready</th>
                <th>Failures</th>
                <th>URL</th>
                <th>Last success</th>
              </tr>
            </thead>
            <tbody>
              {{range .Items}}
              <tr class="env-row" data-name="{{.Name}}" data-namespace="{{.Namespace}}" data-phase="{{.Phase}}">
                <td><a class="row-link" href="#{{.AnchorID}}">{{.Name}}</a><div class="card-ns">{{.Namespace}}</div></td>
                <td><span class="phase">{{.Phase}}</span></td>
                <td>{{.Ready}}</td>
                <td>{{.FailureCount}}</td>
                <td>{{.URL}}</td>
                <td>{{.LastSuccess}}</td>
              </tr>
              {{end}}
            </tbody>
          </table>
        </section>

        <section class="cards">
          {{range .Items}}
          <article class="card env-card" id="{{.AnchorID}}" data-name="{{.Name}}" data-namespace="{{.Namespace}}" data-phase="{{.Phase}}">
            <div class="card-head">
              <div>
                <div class="card-name">{{.Name}}</div>
                <div class="card-ns">{{.Namespace}}</div>
              </div>
              <div class="phase">{{.Phase}}</div>
            </div>

            <div class="meta">
              <div class="meta-box"><strong>Ready</strong>{{.Ready}}</div>
              <div class="meta-box"><strong>Failure count</strong>{{.FailureCount}}</div>
              <div class="meta-box"><strong>URL</strong>{{.URL}}</div>
              <div class="meta-box"><strong>Last error</strong>{{.LastError}}</div>
            </div>

            <h3>Environment Actions</h3>
            <div class="actions">
              <form method="post" action="/action">
                <input type="hidden" name="action" value="diagnose-env">
                <input type="hidden" name="namespace" value="{{.Namespace}}">
                <input type="hidden" name="name" value="{{.Name}}">
                <button type="submit">Diagnose</button>
              </form>
              <form method="post" action="/action">
                <input type="hidden" name="action" value="pause-env">
                <input type="hidden" name="namespace" value="{{.Namespace}}">
                <input type="hidden" name="name" value="{{.Name}}">
                <button type="submit">Pause</button>
              </form>
              <form method="post" action="/action">
                <input type="hidden" name="action" value="resume-env">
                <input type="hidden" name="namespace" value="{{.Namespace}}">
                <input type="hidden" name="name" value="{{.Name}}">
                <button type="submit">Resume</button>
              </form>
              <form method="post" action="/action">
                <input type="hidden" name="action" value="delete-env">
                <input type="hidden" name="namespace" value="{{.Namespace}}">
                <input type="hidden" name="name" value="{{.Name}}">
                <button type="submit" style="border-color: var(--danger); color: var(--danger);">Delete</button>
              </form>
            </div>

            <h3 style="margin-top:18px;">Managed Resources</h3>
            <ul class="list">
              {{range .Resources}}<li>{{.}}</li>{{end}}
            </ul>

            <h3 style="margin-top:18px;">Conditions</h3>
            <div class="conditions">
              {{range .Conditions}}
              <div class="condition">
                <div><strong class="type">{{.Type}}</strong> {{.Status}}</div>
                <div class="reason">Reason: {{.Reason}}</div>
                <div class="message">{{.Message}}</div>
              </div>
              {{end}}
            </div>
          </article>
          {{end}}
        </section>
      </div>
    </section>

    <div class="footer">The console is bound to localhost because it uses your local kube credentials and intentionally limits browser actions to safe terminal-backed Shukra operations instead of exposing a remote shell.</div>
  </div>
  <script>
    (() => {
      const searchInput = document.getElementById("env-filter");
      const phaseSelect = document.getElementById("phase-filter");
      const commandProfile = document.getElementById("command-profile");
      const commandNamespace = document.getElementById("command-namespace");
      const commandName = document.getElementById("command-name");
      const commandHelp = document.getElementById("command-help");
      const rows = Array.from(document.querySelectorAll(".env-row"));
      const cards = Array.from(document.querySelectorAll(".env-card"));

      function matches(el, search, phase) {
        const haystack = [
          el.dataset.name || "",
          el.dataset.namespace || "",
          el.dataset.phase || ""
        ].join(" ").toLowerCase();
        const phaseValue = (el.dataset.phase || "").toLowerCase();
        const searchOk = search === "" || haystack.includes(search);
        const phaseOk = phase === "" || phaseValue === phase;
        return searchOk && phaseOk;
      }

      function applyFilters() {
        const search = (searchInput.value || "").trim().toLowerCase();
        const phase = (phaseSelect.value || "").trim().toLowerCase();

        rows.forEach((row) => {
          row.style.display = matches(row, search, phase) ? "" : "none";
        });
        cards.forEach((card) => {
          card.style.display = matches(card, search, phase) ? "" : "none";
        });
      }

      searchInput.addEventListener("input", applyFilters);
      phaseSelect.addEventListener("change", applyFilters);

      function syncCommandState() {
        if (!commandProfile) return;
        const selected = commandProfile.options[commandProfile.selectedIndex];
        const needsTarget = selected && selected.dataset.needsTarget === "true";
        commandName.disabled = !needsTarget;
        commandName.placeholder = needsTarget ? "basic-app" : "Not required for this command";
        commandName.style.opacity = needsTarget ? "1" : ".55";
        commandHelp.textContent = needsTarget
          ? "This command profile targets a single AppEnvironment. Provide the namespace and environment name before running it."
          : "This command profile works at operator, namespace, or cluster scope. Namespace defaults to the current workspace if you leave it empty.";
      }

      commandProfile.addEventListener("change", syncCommandState);
      syncCommandState();
      applyFilters();
    })();
  </script>
</body>
</html>`))

var consoleActionTemplate = template.Must(template.New("action").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <style>
    body { margin:0; background:#fff; color:#000; font-family:Segoe UI, Arial, sans-serif; }
    .wrap { max-width:1100px; margin:0 auto; padding:24px; }
    .panel { border:1px solid #000; padding:20px; background:#fff; }
    .status { display:inline-block; border:1px solid #000; padding:8px 12px; font-size:12px; text-transform:uppercase; }
    pre { white-space:pre-wrap; word-break:break-word; border:1px solid #000; padding:16px; background:#fff; font-family:Consolas, monospace; }
    a { color:#000; }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="panel">
      <h1>{{.Title}}</h1>
      <p><span class="status">{{if .Success}}Completed{{else}}Failed{{end}}</span></p>
      <p><strong>Command:</strong> {{.Command}}</p>
      <p><strong>Generated:</strong> {{.GeneratedAt}}</p>
      <pre>{{.Output}}</pre>
      <p><a href="/">Back to Web Console</a></p>
    </div>
  </div>
</body>
</html>`))
