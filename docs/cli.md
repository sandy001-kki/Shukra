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
shukra install --oci --chart-version 0.2.0 --operator-namespace shukra-system
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

Use raw YAML editing when:

- you need the full Kubernetes-native spec surface
- you are automating with GitOps
- you want declarative review in pull requests
