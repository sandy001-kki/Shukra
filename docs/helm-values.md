# Helm Values Guide

This guide explains the main Helm values used to install Shukra on local and
production clusters.

The source of truth defaults live in:

- `charts/shukra-operator/values.yaml`

## Image settings

### `image.repository`

The controller image repository.

Default:

```yaml
image:
  repository: ghcr.io/sandy001-kki/shukra-operator
```

Use this when:

- mirroring images into an internal registry
- pinning to a company-controlled repository

### `image.tag`

The image tag to deploy.

Use a fixed release tag in production instead of `latest`.

### `image.pullPolicy`

Usually:

- `IfNotPresent` for stable releases
- `Always` only when you intentionally rely on mutable tags

## Scaling and availability

### `replicaCount`

Number of operator replicas.

Recommended:

- `1` for local development
- `2` or more for production if your cluster and webhook setup are ready for HA

### `leaderElection.enabled`

Should remain enabled in production so multiple replicas do not actively
reconcile at the same time.

### `leaderElection.namespace`

The namespace where leader election Leases are stored.

Usually this should match the operator install namespace.

## Scope control

### `watchNamespace`

Controls what namespaces the operator watches.

Values:

- `""` means all namespaces
- a concrete namespace means single-namespace watch mode

Use a concrete namespace when:

- you want tighter multi-tenant boundaries
- your platform model gives each team its own operator instance

## Performance

### `maxConcurrentReconciles`

Controls how many AppEnvironment reconciles can run at once.

Increase this carefully for:

- larger clusters
- many independent environments

Keep it moderate if:

- the cluster API server is heavily loaded
- you want conservative rollout behavior

## Resource requests and limits

### `resources`

This sets CPU and memory for the operator deployment itself.

Production recommendation:

- always set both requests and limits explicitly
- scale up from defaults if you watch many namespaces or many environments

Example:

```yaml
resources:
  requests:
    cpu: 250m
    memory: 256Mi
  limits:
    cpu: "1"
    memory: 1Gi
```

## Security

### `securityContext.runAsNonRoot`

Should stay `true`.

### `securityContext.runAsUser`

The chart defaults to a non-root UID and should generally remain that way.

## cert-manager integration

### `certmanager.enabled`

Set this to `true` when the chart should manage Issuer and Certificate resources
for the webhook.

### `certmanager.issuerKind`

The kind of issuer resource used for the webhook serving cert.

### `certmanager.issuerName`

The Issuer or ClusterIssuer name used by the chart.

For production, teams often replace the self-signed dev issuer with an
organization-approved issuer.

## Metrics

### `metrics.enabled`

Enable metrics exposure.

Recommended:

- keep enabled in production
- scrape through your standard monitoring stack

### `metrics.port`

The metrics service port.

## Service account

### `serviceAccount.create`

Create a service account as part of the chart.

### `serviceAccount.name`

Use this when your platform wants a pre-created or policy-managed service
account name.

## Production example

```yaml
replicaCount: 2
leaderElection:
  enabled: true
  namespace: shukra-system
watchNamespace: ""
maxConcurrentReconciles: 10
resources:
  requests:
    cpu: 250m
    memory: 256Mi
  limits:
    cpu: "1"
    memory: 1Gi
certmanager:
  enabled: true
  issuerKind: ClusterIssuer
  issuerName: platform-webhook-issuer
metrics:
  enabled: true
serviceAccount:
  create: true
  name: shukra-operator
```

Install with:

```bash
helm upgrade --install shukra-operator oci://ghcr.io/sandy001-kki/charts/shukra-operator \
  --version 0.2.3 \
  -n shukra-system \
  --create-namespace \
  -f values-production.yaml
```

## How to choose values

For local development:

- keep defaults
- use bootstrap or a local chart install

For shared production clusters:

- pin a release version
- set resource requests and limits explicitly
- review watch scope carefully
- use an approved issuer
- run more than one replica when your platform supports it

## Related docs

- [docs/bring-your-own-cluster.md](bring-your-own-cluster.md)
- [docs/troubleshooting.md](troubleshooting.md)
- [docs/tenancy.md](tenancy.md)
