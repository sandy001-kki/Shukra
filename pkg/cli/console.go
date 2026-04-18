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
	Items             []consoleEnvironment
}

type consoleActionPage struct {
	GeneratedAt string
	Title       string
	Command     string
	Output      string
	Success     bool
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
	}

	for _, item := range list.Items {
		env := consoleEnvironment{
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
	case "diagnose-env":
		result.Title = fmt.Sprintf("Diagnose %s/%s", namespace, name)
		result.Command = fmt.Sprintf("shukra diagnose env %s -n %s", name, namespace)
		output, err := diagnoseEnvironmentOutput(ctx, opts, namespace, name)
		result.Output = output
		result.Success = err == nil
	case "pause-env":
		result.Title = fmt.Sprintf("Pause %s/%s", namespace, name)
		result.Command = fmt.Sprintf("shukra env pause %s -n %s", name, namespace)
		output, err := mutateEnvironmentPause(ctx, opts, namespace, name, true)
		result.Output = output
		result.Success = err == nil
	case "resume-env":
		result.Title = fmt.Sprintf("Resume %s/%s", namespace, name)
		result.Command = fmt.Sprintf("shukra env resume %s -n %s", name, namespace)
		output, err := mutateEnvironmentPause(ctx, opts, namespace, name, false)
		result.Output = output
		result.Success = err == nil
	case "delete-env":
		result.Title = fmt.Sprintf("Delete %s/%s", namespace, name)
		result.Command = fmt.Sprintf("shukra env delete %s -n %s", name, namespace)
		output, err := deleteEnvironment(ctx, opts, namespace, name)
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

var consoleTemplate = template.Must(template.New("console").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta http-equiv="refresh" content="15">
  <title>Shukra Web Console</title>
  <style>
    :root { color-scheme: light; }
    * { box-sizing:border-box; }
    body { margin:0; background:#fff; color:#000; font-family:Segoe UI, Arial, sans-serif; }
    .wrap { max-width:1440px; margin:0 auto; padding:24px; }
    .hero, .panel, .card, .table-wrap, .command-box { background:#fff; border:1px solid #000; }
    .hero, .panel, .card { padding:20px; }
    h1, h2, h3 { margin:0; font-weight:700; }
    h1 { font-size:38px; }
    h2 { font-size:22px; margin-bottom:12px; }
    h3 { font-size:16px; margin-bottom:10px; }
    p { line-height:1.5; }
    .sub { margin:10px 0 0; max-width:860px; }
    .grid { display:grid; gap:18px; }
    .top-grid { grid-template-columns:2fr 1fr; margin-top:18px; }
    .stats { display:grid; grid-template-columns:repeat(auto-fit,minmax(160px,1fr)); gap:12px; margin-top:18px; }
    .stat { border:1px solid #000; padding:12px; background:#fff; }
    .label { font-size:12px; text-transform:uppercase; letter-spacing:.08em; }
    .value { font-size:28px; margin-top:8px; }
    .code, pre { margin:0; white-space:pre-wrap; word-break:break-word; font-family:Consolas, monospace; }
    .command-box { padding:14px; }
    .table-wrap { margin-top:18px; overflow:auto; }
    table { width:100%; border-collapse:collapse; }
    th, td { border-bottom:1px solid #000; padding:14px; vertical-align:top; text-align:left; }
    th { background:#fff; font-size:12px; text-transform:uppercase; letter-spacing:.08em; }
    .phase { display:inline-block; border:1px solid #000; padding:5px 10px; font-size:12px; text-transform:uppercase; }
    .cards { display:grid; grid-template-columns:repeat(auto-fit,minmax(430px,1fr)); gap:18px; margin-top:18px; }
    .meta { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:10px; margin:16px 0; }
    .meta-box { border:1px solid #000; padding:10px; min-height:68px; }
    .meta-box strong { display:block; font-size:11px; text-transform:uppercase; margin-bottom:6px; }
    .list { margin:0; padding-left:18px; }
    .list li { margin:4px 0; }
    .conditions { display:grid; gap:10px; margin-top:12px; }
    .condition { border:1px solid #000; padding:10px; }
    .condition .type { font-size:12px; text-transform:uppercase; }
    .condition .reason { font-size:12px; margin-top:6px; }
    .condition .message { font-size:13px; margin-top:6px; }
    .actions { display:flex; gap:10px; flex-wrap:wrap; margin-top:12px; }
    form { margin:0; }
    button, a.button {
      display:inline-block;
      background:#fff;
      color:#000;
      border:1px solid #000;
      padding:10px 14px;
      text-decoration:none;
      cursor:pointer;
      font-size:12px;
      text-transform:uppercase;
      letter-spacing:.08em;
    }
    .footer { font-size:13px; margin-top:18px; }
    @media (max-width: 980px) {
      .top-grid { grid-template-columns:1fr; }
      .cards { grid-template-columns:1fr; }
    }
  </style>
</head>
<body>
  <div class="wrap">
    <section class="hero">
      <h1>Shukra Web Console</h1>
      <p class="sub">A localhost-only operations dashboard for AppEnvironment health, operator state, safe browser-triggered actions, and cluster troubleshooting. The console stays on <strong>{{.LocalhostAddress}}</strong> so it can use your local kube context without becoming an exposed remote admin surface.</p>
      <div class="stats">
        <div class="stat"><div class="label">Visible environments</div><div class="value">{{.Count}}</div></div>
        <div class="stat"><div class="label">Ready</div><div class="value">{{.ReadyCount}}</div></div>
        <div class="stat"><div class="label">Running</div><div class="value">{{.RunningCount}}</div></div>
        <div class="stat"><div class="label">Paused</div><div class="value">{{.PausedCount}}</div></div>
        <div class="stat"><div class="label">Failed</div><div class="value">{{.FailedCount}}</div></div>
        <div class="stat"><div class="label">Degraded</div><div class="value">{{.DegradedCount}}</div></div>
      </div>
    </section>

    <section class="grid top-grid">
      <div class="panel">
        <h2>Cluster Summary</h2>
        <div class="command-box"><pre class="code">Cluster context: {{.Cluster}}
Namespace flag: {{.Namespace}}
Operator namespace: {{.OperatorNamespace}}
Generated: {{.GeneratedAt}}
JSON API: /api/environments</pre></div>
        <div class="actions">
          <a class="button" href="/">Refresh</a>
          <a class="button" href="/api/environments">Open JSON API</a>
        </div>
      </div>

      <div class="panel">
        <h2>Operator Actions</h2>
        <p>These buttons run safe, whitelisted local actions. The console does not expose arbitrary shell access.</p>
        <div class="actions">
          <form method="post" action="/action"><input type="hidden" name="action" value="doctor"><button type="submit">Run Doctor</button></form>
          <form method="post" action="/action"><input type="hidden" name="action" value="diagnose-operator"><button type="submit">Diagnose Operator</button></form>
          <form method="post" action="/action"><input type="hidden" name="action" value="operator-logs"><button type="submit">Tail Operator Logs</button></form>
          <form method="post" action="/action"><input type="hidden" name="action" value="apply-basic"><button type="submit">Apply Basic Example</button></form>
        </div>
      </div>
    </section>

    <section class="panel" style="margin-top:18px;">
      <h2>Operator Pods</h2>
      <ul class="list">
        {{range .OperatorPods}}
        <li><strong>{{.Name}}</strong> phase={{.Status}} node={{.Node}}</li>
        {{end}}
      </ul>
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
          <tr>
            <td><strong>{{.Name}}</strong><div>{{.Namespace}}</div></td>
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
      <article class="card">
        <div style="display:flex;justify-content:space-between;gap:12px;align-items:flex-start;">
          <div>
            <div style="font-size:24px;font-weight:700;">{{.Name}}</div>
            <div>{{.Namespace}}</div>
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
            <button type="submit">Delete</button>
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

    <div class="footer">The console is intentionally bound to localhost because it uses your local kube credentials and allows only safe, whitelisted Shukra actions.</div>
  </div>
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
