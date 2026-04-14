# Shukra Operator

Shukra Operator — One YAML. Complete Environment.

Shukra Operator is a production-grade Kubernetes Operator that lets users define
an entire application environment through a single `AppEnvironment` custom
resource. Instead of maintaining many separate Kubernetes manifests, a user
declares the desired environment once and Shukra continuously reconciles the
required resources for them.

Repository: [github.com/sandy001-kki/Shukra](https://github.com/sandy001-kki/Shukra)

## Why this project exists

Running an application in Kubernetes normally requires several objects:

- a `Deployment` for Pods
- a `Service` for internal networking
- a `ConfigMap` for configuration
- one or more `Secret` references
- an `Ingress` for external access
- `HorizontalPodAutoscaler` rules
- migration and restore `Job` objects
- backup `CronJob` objects
- `NetworkPolicy`
- `PodDisruptionBudget`

Managing those objects by hand becomes repetitive and error-prone. Shukra raises
the abstraction level. Users describe the application and its operational needs,
and the operator turns that into the lower-level Kubernetes resources.

## What Shukra manages

Depending on the `AppEnvironment` spec, Shukra can create and manage:

- `Deployment`
- `Service`
- `ConfigMap`
- references to existing `Secret` objects
- `Ingress`
- `HorizontalPodAutoscaler`
- migration `Job`
- restore `Job`
- backup `CronJob`
- `NetworkPolicy`
- `PodDisruptionBudget`

On deletion, Shukra runs finalizer-driven cleanup before releasing the custom
resource.

## Main features

- One high-level `AppEnvironment` API for full environment lifecycle management
- Explicit API versioning with `v1alpha1` to `v1beta1` conversion
- Mutating, validating, and conversion webhooks
- Idempotent reconciliation using controller-runtime
- Finalizers for safe deletion workflows
- Status conditions and phase reporting on the custom resource
- Structured logging, Kubernetes events, and Prometheus metrics
- Leader election for production-safe multi-replica deployments
- Helm chart, generated manifests, examples, and GitHub Actions workflows

## Architecture overview

```text
AppEnvironment
  -> mutating webhook
  -> validating webhook
  -> conversion webhook
  -> Shukra controller
  -> ordered child-resource reconciliation
  -> status, conditions, events, metrics
```

High-level reconcile order:

1. Fetch `AppEnvironment`
2. Handle deletion and finalizers
3. Respect paused mode
4. Validate dependencies and tenancy rules
5. Reconcile ConfigMap, Service, Deployment, HPA, migration, restore, Ingress,
   NetworkPolicy, PDB, and backup CronJob
6. Update status and conditions

More detail is available in [docs/architecture.md](docs/architecture.md).

## Who should use it

Shukra is useful for:

- application developers who want fewer Kubernetes details
- platform teams enforcing a standard app deployment contract
- DevOps and SRE teams that want safer lifecycle automation
- organizations deploying many similar services across namespaces

## Prerequisites

- Go 1.21
- Kubernetes 1.26+
- Helm 3.13+
- cert-manager 1.13+
- Docker and kind for local cluster testing

## Quick install from GitHub checkout

Clone the repository:

```bash
git clone https://github.com/sandy001-kki/Shukra.git
cd Shukra
```

Install the operator into a cluster:

```bash
helm install shukra-operator charts/shukra-operator \
  -n shukra-system \
  --create-namespace
```

That chart installs:

- the `AppEnvironment` CRD
- the controller Deployment
- RBAC
- webhook configuration
- cert-manager issuer and certificate resources
- metrics and webhook Services

## Five minute quickstart

If you are new to the project, this is the shortest useful path:

```bash
git clone https://github.com/sandy001-kki/Shukra.git
cd Shukra
helm install shukra-operator charts/shukra-operator -n shukra-system --create-namespace
kubectl apply -f examples/basic.yaml
kubectl get appenvironment basic-app -n default -o yaml
kubectl get deploy,svc,cm,pods -n default
```

If you want a more beginner-friendly walkthrough, read
[docs/getting-started.md](docs/getting-started.md).

## One-command local bootstrap

If you want Shukra to set up a complete local workflow for you on Windows, run:

```powershell
powershell -ExecutionPolicy Bypass -File .\hack\bootstrap-local.ps1
```

That script will:

- start Docker Desktop
- create a kind cluster
- install cert-manager
- build the operator image
- load the image into kind
- install the Shukra Helm chart
- apply `examples/basic.yaml`
- wait for the sample Deployment rollout

There is also a matching Make target:

```bash
make bootstrap-local
```

## Install from OCI Helm chart

If a published chart is available in GHCR, install it directly:

```bash
helm install shukra-operator oci://ghcr.io/sandy001-kki/charts/shukra-operator \
  --version 0.1.0 \
  -n shukra-system \
  --create-namespace
```

## First user workflow

After the operator is installed, a new user only needs to apply an
`AppEnvironment`.

Use the basic example:

```bash
kubectl apply -f examples/basic.yaml
kubectl get appenvironments.apps.shukra.io
kubectl describe appenvironment basic-app
kubectl get deploy,svc,cm -n default
```

The shipped basic example is:

```yaml
apiVersion: apps.shukra.io/v1beta1
kind: AppEnvironment
metadata:
  name: basic-app
spec:
  app:
    image: nginx:1.27
    containerPort: 80
    livenessProbe:
      httpGet:
        path: /
        port: 80
      initialDelaySeconds: 10
    readinessProbe:
      httpGet:
        path: /
        port: 80
      initialDelaySeconds: 5
    replicas: 2
    resources:
      requests:
        cpu: 100m
        memory: 128Mi
      limits:
        cpu: 500m
        memory: 256Mi
  service:
    enabled: true
```

What happens when that file is applied:

- Shukra validates the spec
- adds defaults where needed
- creates a `ConfigMap`
- creates a `Service`
- creates a `Deployment`
- updates `status.phase`, `status.conditions`, and child resource names

## What a user edits

Users mainly work with the `spec` fields on `AppEnvironment`.

Key sections include:

- `spec.app`
  image, replicas, ports, probes, environment, resources, secret references
- `spec.config`
  key-value app configuration
- `spec.service`
  internal traffic exposure
- `spec.ingress`
  external hostname and routing
- `spec.database`
  database mode and referenced secret
- `spec.migration`
  migration job configuration and `migrationID`
- `spec.restore`
  restore workflow and `triggerNonce`
- `spec.backup`
  backup scheduling and destination
- `spec.autoscaling`
  HPA behavior
- `spec.security`
  network policy, PDB, security context
- `spec.paused`
  stop mutations while still refreshing status

## Examples included in this repo

- [examples/basic.yaml](examples/basic.yaml)
  Smallest useful app environment
- [examples/ingress.yaml](examples/ingress.yaml)
  App with external ingress configuration
- [examples/autoscaling.yaml](examples/autoscaling.yaml)
  App with HPA configuration
- [examples/migration.yaml](examples/migration.yaml)
  App with database and migration workflow
- [examples/restore.yaml](examples/restore.yaml)
  App with backup and restore flow
- [examples/paused.yaml](examples/paused.yaml)
  App with reconciliation paused

## How users observe their environment

The `AppEnvironment` status is the first place to inspect.

Important status fields include:

- `status.phase`
- `status.conditions`
- `status.childResources`
- `status.lastError`
- `status.failureCount`
- `status.lastAppliedMigrationID`
- `status.lastProcessedRestoreNonce`

Useful commands:

```bash
kubectl get appenvironment <name> -o yaml
kubectl describe appenvironment <name>
kubectl get deploy,svc,ingress,hpa,job,cronjob -n <namespace>
kubectl logs -n shukra-system deploy/shukra-operator
```

## Backup and restore

Backups are declared in `spec.backup` and materialize as a managed `CronJob`.
Restore runs are controlled by `spec.restore.triggerNonce`. Shukra only creates
a new restore `Job` when the nonce changes, which makes restore execution
intentional and idempotent.

## Namespace tenancy and secret model

Shukra treats a Kubernetes namespace as the tenant boundary.

Rules enforced by the operator:

- secret references must stay in the same namespace
- service account references must stay in the same namespace
- Shukra never creates secret objects from inline values
- Shukra never logs secret values
- ingress host uniqueness is enforced cluster-wide

This makes the operator compatible with secret managers such as External
Secrets Operator, as long as the materialized Secret exists in the same
namespace.

See [docs/tenancy.md](docs/tenancy.md) for the full model.

## Local development workflow

Common workflow:

```bash
make generate
make manifests
make test
make run
```

Other useful tasks:

```bash
make lint
make docker-build
make helm-package
make docs-generate
```

## Local cluster workflow

For contributors who want to run everything end-to-end locally:

1. Start Docker Desktop
2. Create a kind cluster
3. Install cert-manager
4. Build the operator image
5. Load the image into kind
6. Install the Shukra chart
7. Apply an example manifest

This repository has been validated through that flow locally.

## Release and versioning policy

Shukra follows semantic versioning.

Git tags like `v0.1.0` drive:

- the container image tag
- the Helm chart `version`
- the Helm chart `appVersion`

Charts are published as OCI artifacts to:

`oci://ghcr.io/sandy001-kki/charts/shukra-operator`

## CI and release

The repository contains:

- `.github/workflows/ci.yaml`
  lint, vet, generate, and test validation
- `.github/workflows/release.yaml`
  multi-arch image build, Helm packaging, and release publishing

## Documentation

- [docs/getting-started.md](docs/getting-started.md)
- [docs/api.md](docs/api.md)
- [docs/architecture.md](docs/architecture.md)
- [docs/tenancy.md](docs/tenancy.md)
- [docs/troubleshooting.md](docs/troubleshooting.md)

The API reference is generated by `make docs-generate` using `crd-ref-docs`.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for branch workflow, generation
requirements, and test expectations.

## In one sentence

Shukra Operator lets a user describe an application environment once and have
Kubernetes continuously build, maintain, and clean up the underlying runtime
resources for them.
