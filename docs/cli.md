# Shukra CLI

This guide explains the `shukra` CLI that ships alongside the operator.

## Why the CLI exists

The Kubernetes Operator is the control plane. It watches `AppEnvironment`
resources and reconciles them into real cluster resources.

The CLI is the user-facing interface. It helps people:

- install the operator
- bootstrap a local cluster
- generate starter manifests
- inspect environment status
- pause or resume reconciliation
- request migrations and restores

The CLI does not replace the operator. It makes the operator easier to use.

## Chat mode

Shukra also includes an English-first chat mode for PowerShell users who want a
more guided interface.

Open the interactive assistant:

```powershell
shukra chat
```

Run one English command and exit:

```powershell
shukra chat --message "status basic-app"
```

Example phrases:

- `status basic-app`
- `list environments`
- `show resources for basic-app`
- `diagnose basic-app`
- `show operator status`
- `apply examples/basic.yaml`
- `show operator logs`
- `pause basic-app`
- `resume basic-app`
- `delete basic-app`
- `install operator from oci version 0.2.3`
- `bootstrap local`

## Doctor command

Use the doctor command to check whether the local Shukra environment is ready.

```powershell
shukra doctor
shukra doctor --output json
```

It checks:

- Docker CLI presence
- Docker engine responsiveness
- `kubectl` availability
- `helm` availability
- kubeconfig loading
- Kubernetes API reachability
- AppEnvironment API availability
- operator pod readiness
- cert-manager pod readiness

On existing remote clusters, Docker is optional. `shukra doctor` treats Docker
as required only for local image-build and kind-style workflows.

The JSON mode is useful for CI pipelines, scripted environment checks, and
bring-your-own-cluster readiness validation before install or upgrade work.

## Diagnose commands

If you prefer explicit commands over chat phrasing, use the diagnose surface:

```powershell
shukra diagnose env basic-app -n default
shukra diagnose operator
```

`diagnose env` gives a focused summary of phase, readiness, paused state,
failure count, and next checks. `diagnose operator` shows the current operator
Pod health and placement.

## Ask command

Use `shukra ask` for grounded answers from the local Shukra documentation and
examples when you want help without opening the full chat interface.

```powershell
shukra ask "How do I install Shukra on my own cluster?"
shukra ask "How do I install Shukra on EKS?" --top 5
shukra ask "How do backups work?" --output json
```

This mode does not depend on an API key or external model. It searches the repo
docs and examples and returns the best grounded snippets plus the matching
source files.

## Shell completion

The CLI can generate shell completions, including PowerShell completions for
Windows users:

```powershell
shukra completion powershell
```

Typical PowerShell setup:

```powershell
shukra completion powershell | Out-String | Invoke-Expression
```

For persistent local setup, add that line to your PowerShell profile.

## Web Console

Shukra includes a lightweight local Web Console for users who want a browser
dashboard instead of terminal-only inspection.

```powershell
shukra console
```

By default it listens on `127.0.0.1:8088`.

The console is intentionally localhost-only by default because it reads your
current kube context and is meant to stay inside your local trust boundary.
That keeps the browser UI useful without turning it into an exposed remote
cluster control surface.

What the console includes today:

- environment table and per-environment cards
- operator pod status
- JSON API at `/api/environments`
- a browser Command Center for safe terminal-backed command profiles
- safe action buttons and command profiles for:
  - `doctor`
  - operator diagnosis
  - operator logs
  - apply basic example
  - cluster AppEnvironment listing
  - node listing
  - namespace pods, services, ConfigMaps, jobs, and CronJobs
  - environment summary, YAML, describe, and managed-resource listing
  - environment diagnosis
  - pause
  - resume
  - delete

The console does not provide arbitrary shell access from the browser. It runs
only whitelisted local Shukra and `kubectl` actions and returns their output in
the page.

## Core commands

Print the CLI version:

```bash
shukra version
```

Install the operator from the local chart:

```bash
shukra install --operator-namespace shukra-system
```

Install the operator from the published OCI chart:

```bash
shukra install --oci --chart-version 0.2.3 --operator-namespace shukra-system
```

Run machine-readable health checks:

```bash
shukra doctor --output json
```

Run direct diagnosis commands:

```bash
shukra diagnose env basic-app -n default
shukra diagnose operator
```

Ask grounded questions from local docs:

```bash
shukra ask "How do I install Shukra on AKS?"
```

Bootstrap a local Windows development cluster:

```powershell
shukra bootstrap local
```

Generate a starter environment manifest:

```bash
shukra env init my-app --image nginx:1.27 --output my-app.yaml
```

Apply a manifest:

```bash
shukra env apply -f my-app.yaml
```

Show a summary status:

```bash
shukra env status my-app -n default
```

Show the full resource as YAML:

```bash
shukra env status my-app -n default -o yaml
```

Pause and resume reconciliation:

```bash
shukra env pause my-app -n default
shukra env resume my-app -n default
```

Request a migration:

```bash
shukra env migrate my-app -n default --migration-id v2-add-indexes --image busybox:1.36
```

Request a restore:

```bash
shukra env restore my-app -n default \
  --trigger-nonce restore-20260414-001 \
  --image busybox:1.36 \
  --source "echo restoring && sleep 5"
```

Delete an environment:

```bash
shukra env delete my-app -n default
```

## Release assets

The GitHub release pipeline publishes CLI binaries alongside the operator chart.

Current release asset names:

- `shukra-linux-amd64`
- `shukra-windows-amd64.exe`
- `shukra-darwin-arm64`

## Recommended use

Use the CLI for:

- day-one installation
- day-two environment inspection
- common lifecycle actions
- starter YAML generation

If you already have a Kubernetes cluster, combine this guide with:

- [docs/bring-your-own-cluster.md](bring-your-own-cluster.md)
- [docs/helm-values.md](helm-values.md)

Use raw YAML editing when:

- you need the full Kubernetes-native spec surface
- you are automating with GitOps
- you want declarative review in pull requests
