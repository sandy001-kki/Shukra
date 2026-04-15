# Bring Your Own Cluster

This guide is for users who already have a Kubernetes cluster and want to run
Shukra there.

Shukra is not tied to Docker Desktop or kind. Those are only local development
paths. The operator is intended to run on general Kubernetes clusters as long as
the cluster meets the documented version and admission requirements.

## Supported cluster styles

Typical supported environments include:

- kind
- Minikube
- k3d
- k3s
- EKS
- GKE
- AKS
- other conformant Kubernetes clusters

## What you need

- Kubernetes 1.26+
- `kubectl` access to the target cluster
- Helm 3.13+
- cert-manager 1.13+ or an equivalent way to manage webhook serving certificates

## Step 1: confirm cluster access

Make sure your current context points at the intended cluster:

```bash
kubectl config current-context
kubectl get nodes
```

You can also use:

```bash
shukra doctor
```

The doctor command is cluster-aware. It checks Kubernetes connectivity on any
supported cluster and treats Docker as optional unless you are clearly using a
local image-build workflow.

## Step 2: install cert-manager

If your platform already manages webhook certificates another way, adapt the
chart inputs accordingly. Otherwise, install cert-manager first.

Example:

```bash
helm repo add jetstack https://charts.jetstack.io
helm repo update
helm upgrade --install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --version v1.17.4 \
  --set crds.enabled=true
```

## Step 3: install Shukra

Install from the published OCI chart:

```bash
helm install shukra-operator oci://ghcr.io/sandy001-kki/charts/shukra-operator \
  --version 0.2.3 \
  -n shukra-system \
  --create-namespace
```

Or install from a Git checkout:

```bash
helm install shukra-operator charts/shukra-operator \
  -n shukra-system \
  --create-namespace
```

## Step 4: choose production values

At minimum, review:

- `replicaCount`
- `leaderElection.namespace`
- `watchNamespace`
- `resources`
- `serviceAccount`
- `certmanager`
- `metrics`

See [docs/helm-values.md](helm-values.md) for the detailed values guide.

## Step 5: verify the operator

```bash
kubectl get pods -n shukra-system
kubectl logs -n shukra-system deploy/shukra-operator
shukra doctor
```

Expected outcome:

- the operator pod is running
- the `AppEnvironment` API is reachable
- cert-manager is ready

## Step 6: apply an environment

```bash
kubectl apply -f examples/basic.yaml
```

Then inspect:

```bash
kubectl get appenvironment basic-app -n default -o yaml
shukra chat --message "status basic-app"
shukra chat --message "diagnose basic-app"
```

## Notes for managed clusters

### EKS

- ensure IAM and CNI policies allow normal ingress/networking behavior for your app
- expose metrics and ingress according to your organization’s standards

### GKE

- review ingress class settings
- make sure cert-manager and webhook networking are allowed by cluster policy

### AKS

- validate network policy compatibility with your chosen CNI mode
- confirm Pod security and RBAC restrictions allow the operator namespace to function

## Notes for local clusters

For kind, Minikube, k3d, and similar local environments, you may still prefer
the bootstrap script. That is a convenience workflow, not a Shukra
requirement.

## When Docker is required

Docker is required when:

- you build the operator image locally
- you use kind or another Docker-backed local cluster
- you use the one-command bootstrap workflow

Docker is not required when:

- you are installing the published OCI chart into an existing remote cluster
- you are using the published GHCR image as-is

## Next reads

- [docs/helm-values.md](helm-values.md)
- [docs/troubleshooting.md](troubleshooting.md)
- [docs/tenancy.md](tenancy.md)
- [docs/cli.md](cli.md)
