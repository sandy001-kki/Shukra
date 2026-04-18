// This file implements a lightweight Shukra Web Console. It exists to give
// users a fast visual dashboard for live AppEnvironment status without needing
// a separate frontend build or backend service.
package cli

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
	"github.com/spf13/cobra"
)

type consoleEnvironment struct {
	Name         string
	Namespace    string
	Phase        string
	URL          string
	Ready        string
	FailureCount int32
	LastError    string
	Resources    []string
}

type consolePage struct {
	GeneratedAt string
	Cluster     string
	Namespace   string
	Count       int
	Items       []consoleEnvironment
}

func newConsoleCommand(opts *RootOptions) *cobra.Command {
	var addr string

	cmd := &cobra.Command{
		Use:   "console",
		Short: "Run a lightweight local Shukra Web Console",
		RunE: func(cmd *cobra.Command, args []string) error {
			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
				defer cancel()

				page, err := buildConsolePage(ctx, opts)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if err := consoleTemplate.Execute(w, page); err != nil {
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

func buildConsolePage(ctx context.Context, opts *RootOptions) (*consolePage, error) {
	kubeClient, _, err := buildClient(ctx, opts)
	if err != nil {
		return nil, err
	}

	var list appsv1beta1.AppEnvironmentList
	if err := kubeClient.List(ctx, &list); err != nil {
		return nil, fmt.Errorf("list AppEnvironments: %w", err)
	}

	items := make([]consoleEnvironment, 0, len(list.Items))
	for _, item := range list.Items {
		items = append(items, consoleEnvironment{
			Name:         item.Name,
			Namespace:    item.Namespace,
			Phase:        emptyDash(item.Status.Phase),
			URL:          emptyDash(item.Status.URL),
			Ready:        consoleConditionStatus(item.Status.Conditions, appsv1beta1.ConditionReady),
			FailureCount: item.Status.FailureCount,
			LastError:    emptyDash(item.Status.LastError),
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
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Namespace == items[j].Namespace {
			return items[i].Name < items[j].Name
		}
		return items[i].Namespace < items[j].Namespace
	})

	return &consolePage{
		GeneratedAt: time.Now().Format(time.RFC1123),
		Cluster:     currentContext(opts),
		Namespace:   opts.Namespace,
		Count:       len(items),
		Items:       items,
	}, nil
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

var consoleTemplate = template.Must(template.New("console").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Shukra Web Console</title>
  <style>
    :root { color-scheme: dark; --bg:#0b1220; --panel:#111c2f; --muted:#93a4bf; --text:#f8fafc; --accent:#38bdf8; --good:#22c55e; --warn:#f59e0b; --bad:#ef4444; }
    * { box-sizing:border-box; }
    body { margin:0; font-family:Segoe UI, system-ui, sans-serif; background:radial-gradient(circle at top right,#123b3a 0,#0b1220 45%); color:var(--text); }
    .wrap { max-width:1200px; margin:0 auto; padding:32px 20px 40px; }
    .hero { background:linear-gradient(135deg, rgba(17,28,47,.95), rgba(13,80,79,.8)); border:1px solid rgba(148,163,184,.15); border-radius:24px; padding:28px; box-shadow:0 20px 60px rgba(0,0,0,.35); }
    h1 { margin:0 0 8px; font-size:42px; }
    .sub { margin:0; color:var(--muted); font-size:18px; }
    .stats { display:grid; grid-template-columns:repeat(auto-fit,minmax(180px,1fr)); gap:16px; margin:24px 0; }
    .stat, .card { background:rgba(15,23,42,.82); border:1px solid rgba(148,163,184,.14); border-radius:18px; padding:18px; }
    .label { color:var(--muted); font-size:13px; text-transform:uppercase; letter-spacing:.08em; }
    .value { font-size:28px; font-weight:700; margin-top:8px; }
    .grid { display:grid; grid-template-columns:repeat(auto-fit,minmax(320px,1fr)); gap:18px; }
    .title { display:flex; justify-content:space-between; gap:12px; align-items:flex-start; }
    .name { font-size:24px; font-weight:700; }
    .ns { color:var(--muted); font-size:14px; margin-top:4px; }
    .pill { border-radius:999px; padding:6px 12px; font-size:12px; font-weight:700; }
    .running { background:rgba(34,197,94,.15); color:#86efac; }
    .configuring { background:rgba(56,189,248,.15); color:#7dd3fc; }
    .failed { background:rgba(239,68,68,.18); color:#fca5a5; }
    .paused { background:rgba(245,158,11,.18); color:#fcd34d; }
    .meta { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:12px; margin:18px 0; }
    .meta div { background:rgba(15,23,42,.55); border-radius:12px; padding:12px; }
    .meta strong { display:block; color:var(--muted); font-size:12px; margin-bottom:6px; }
    ul { margin:12px 0 0; padding-left:18px; color:#dbeafe; }
    .footer { margin-top:24px; color:var(--muted); font-size:14px; }
    a { color:#7dd3fc; text-decoration:none; }
  </style>
</head>
<body>
  <div class="wrap">
    <section class="hero">
      <h1>Shukra Web Console</h1>
      <p class="sub">Live AppEnvironment overview from your cluster.</p>
      <div class="stats">
        <div class="stat"><div class="label">Cluster context</div><div class="value">{{.Cluster}}</div></div>
        <div class="stat"><div class="label">Visible environments</div><div class="value">{{.Count}}</div></div>
        <div class="stat"><div class="label">Namespace flag</div><div class="value">{{.Namespace}}</div></div>
      </div>
    </section>
    <div class="grid">
      {{range .Items}}
      <section class="card">
        <div class="title">
          <div>
            <div class="name">{{.Name}}</div>
            <div class="ns">{{.Namespace}}</div>
          </div>
          <div class="pill {{if eq .Phase "Running"}}running{{else if eq .Phase "Paused"}}paused{{else if eq .Phase "Failed"}}failed{{else}}configuring{{end}}">{{.Phase}}</div>
        </div>
        <div class="meta">
          <div><strong>Ready</strong>{{.Ready}}</div>
          <div><strong>Failure count</strong>{{.FailureCount}}</div>
          <div><strong>URL</strong>{{.URL}}</div>
          <div><strong>Last error</strong>{{.LastError}}</div>
        </div>
        <strong>Managed resources</strong>
        <ul>
          {{range .Resources}}<li>{{.}}</li>{{end}}
        </ul>
      </section>
      {{end}}
    </div>
    <div class="footer">Generated {{.GeneratedAt}}. Refresh the page to pull current cluster state.</div>
  </div>
</body>
</html>`))
