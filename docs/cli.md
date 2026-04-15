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
