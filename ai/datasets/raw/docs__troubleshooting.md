# Troubleshooting

## Webhook TLS issues

Shukra expects cert-manager to inject the CA bundle into the mutating and
validating webhook configurations. If cert-manager is not available in your
cluster, you can still run the operator by generating a serving certificate,
creating the webhook TLS Secret, and patching the CA bundle into the webhook
configurations manually before starting the manager.

Symptoms usually include webhook connection failures, x509 errors, or rejected
admission requests even when the controller Pod is healthy.

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

## How to read operator logs

Every log line is structured with:

- `namespace`
- `name`
- `generation`
- `reconcileID`

This makes it possible to follow one reconcile attempt across validation,
resource creation, retries, and final status writes.

