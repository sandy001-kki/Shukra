# Troubleshooting

## Start with doctor

Before diving into logs, run:

```bash
shukra doctor
```

For bring-your-own-cluster users, this is the fastest way to confirm:

- kubeconfig resolution
- cluster API reachability
- AppEnvironment API availability
- operator pod readiness
- cert-manager readiness

Docker warnings in `shukra doctor` are informational on remote clusters unless
you are intentionally using a local image-build workflow.

## Webhook TLS issues

Shukra expects cert-manager to inject the CA bundle into the mutating and
validating webhook configurations. If cert-manager is not available in your
cluster, you can still run the operator by generating a serving certificate,
creating the webhook TLS Secret, and patching the CA bundle into the webhook
configurations manually before starting the manager.

Symptoms usually include webhook connection failures, x509 errors, or rejected
admission requests even when the controller Pod is healthy.

On managed clusters, also verify:

- the webhook service is reachable from the API server
- network policy is not blocking admission traffic
- your chosen issuer is valid for the operator namespace

## Migration not running

Migration Jobs are keyed by `spec.migration.migrationID`. If the migration ID
did not change, the operator assumes that migration request has already been
processed. To intentionally trigger another migration, change the
`migrationID`.

## Restore not triggering

Restore Jobs are keyed by `spec.restore.triggerNonce`. If you reapply the same
manifest with the same nonce, the operator treats it as already processed. To
launch another restore run, set a new nonce value.

## Phase stuck in Degraded

Inspect:

- `status.conditions`
- `status.lastError`
- the child resource list in `status.childResources`
- operator logs for the same `reconcileID`

The degraded state usually means at least one child resource failed to
reconcile or an external dependency such as a referenced Secret is missing.

## Common RBAC errors

- Missing `leases` permission breaks leader election
- Missing `events` permission removes lifecycle event visibility
- Missing `jobs` or `cronjobs` permissions blocks migration, restore, and backup flows
- Missing `ingresses` or `networkpolicies` permissions prevents network reconciliation

## Bring-your-own-cluster checks

If you are not using kind or local Docker workflows, confirm:

- the current `kubectl` context targets the intended cluster
- the `shukra-system` namespace exists or Helm can create it
- cert-manager is installed or webhook TLS is otherwise managed
- cluster policy allows webhook traffic and leader-election Leases
- the published GHCR image is pullable from the cluster

## How to read operator logs

Every log line is structured with:

- `namespace`
- `name`
- `generation`
- `reconcileID`

This makes it possible to follow one reconcile attempt across validation,
resource creation, retries, and final status writes.
