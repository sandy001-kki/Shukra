# Getting Started

This guide is for a brand new user who wants to run Shukra Operator with the
smallest amount of Kubernetes knowledge.

It covers both:

- local development clusters
- users who already have a Kubernetes cluster

## What you need

- a Kubernetes cluster
- `kubectl`
- Helm
- cert-manager installed in the cluster

If you already have a cluster, you can skip the local bootstrap path and follow
[docs/bring-your-own-cluster.md](bring-your-own-cluster.md).

## Install the operator

```bash
helm install shukra-operator charts/shukra-operator \
  -n shukra-system \
  --create-namespace
```

If you want the published chart instead of a Git checkout:

```bash
helm install shukra-operator oci://ghcr.io/sandy001-kki/charts/shukra-operator \
  --version 0.2.3 \
  -n shukra-system \
  --create-namespace
```

## One-command local workflow on Windows

If you want the repository to do the local setup for you, run:

```powershell
powershell -ExecutionPolicy Bypass -File .\hack\bootstrap-local.ps1
```

This automates Docker startup, kind cluster creation, cert-manager install,
operator image build, image load, Helm install, and applying the basic example.

This path is for local development convenience only.

## Apply your first environment

Use the working basic example:

```bash
kubectl apply -f examples/basic.yaml
```

## Check what Shukra created

```bash
kubectl get appenvironment basic-app -n default -o yaml
kubectl get deploy,svc,cm -n default
kubectl get pods -n default
```

Expected outcome:

- one `AppEnvironment`
- one `Deployment`
- one `Service`
- one `ConfigMap`
- running application Pods

## Update your application

Change the `AppEnvironment` YAML, then apply it again:

```bash
kubectl apply -f examples/basic.yaml
```

Shukra will reconcile the existing environment instead of creating a second one.

## Delete your environment

```bash
kubectl delete -f examples/basic.yaml
```

Shukra will run its finalizer flow and clean up owned resources before the
custom resource disappears.

## Next examples

- `examples/ingress.yaml`
- `examples/autoscaling.yaml`
- `examples/migration.yaml`
- `examples/restore.yaml`
- `examples/paused.yaml`

## If something goes wrong

```bash
shukra doctor
kubectl describe appenvironment basic-app
kubectl logs -n shukra-system deploy/shukra-operator
```

For more detail, read:

- `docs/bring-your-own-cluster.md`
- `docs/learning-path.md`
- `docs/architecture.md`
- `docs/tenancy.md`
- `docs/troubleshooting.md`
