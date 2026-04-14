# Migration And Restore Walkthrough

This guide explains how to use Shukra for migration and restore workflows in a
practical way.

## What migration means here

A migration is a one-time job that changes database structure or seed data.

Shukra models migration as a Kubernetes `Job` controlled by:

- `spec.migration.enabled`
- `spec.migration.image`
- `spec.migration.migrationID`

The important idempotency rule is:

- change `migrationID` when you want a new migration request

## What restore means here

A restore is a recovery action triggered on demand.

Shukra models restore as a Kubernetes `Job` controlled by:

- `spec.restore.enabled`
- `spec.restore.image`
- `spec.restore.triggerNonce`

The important idempotency rule is:

- change `triggerNonce` when you want a new restore request

## Before you begin

For migration examples using an external database secret, create a demo secret:

```bash
kubectl create secret generic migration-db \
  -n default \
  --from-literal=username=demo \
  --from-literal=password=demo \
  --from-literal=database=app
```

## Migration example

Apply the migration example:

```bash
kubectl apply -f examples/migration.yaml
```

Check the result:

```bash
kubectl get appenvironment migration-app -n default -o yaml
kubectl get jobs -n default
kubectl logs job/migration-app-migration-v1-initial-schema -n default
```

What you should see:

- a migration `Job`
- `status.lastAppliedMigrationID` set on the custom resource
- `MigrationReady` condition updated

## Trigger a second migration

Edit the manifest and change:

```yaml
migrationID: v2-add-indexes
```

Then apply again:

```bash
kubectl apply -f examples/migration.yaml
```

That creates a new migration request because the idempotency key changed.

## Restore example

For the local demo example, the restore container uses a simple shell command so
the job is easy to observe on any cluster. In a real environment, the restore
image should implement the actual restore logic for your backup source.

Apply the restore example:

```bash
kubectl apply -f examples/restore.yaml
```

Check the result:

```bash
kubectl get appenvironment restore-app -n default -o yaml
kubectl get jobs -n default
kubectl logs job/restore-app-restore-restore-20240101-001 -n default
```

What you should see:

- a restore `Job`
- `status.lastProcessedRestoreNonce` set on the custom resource
- `RestoreReady` condition updated

## Trigger another restore

Edit the manifest and change:

```yaml
triggerNonce: restore-20240101-002
```

Apply again:

```bash
kubectl apply -f examples/restore.yaml
```

That tells Shukra to create a new restore job because the restore idempotency
key changed.

## How to inspect status

Useful fields on the custom resource:

- `status.lastAppliedMigrationID`
- `status.lastProcessedRestoreNonce`
- `status.conditions`
- `status.lastError`
- `status.phase`

Useful commands:

```bash
kubectl describe appenvironment migration-app -n default
kubectl describe appenvironment restore-app -n default
kubectl get jobs -n default
kubectl logs -n shukra-system deploy/shukra-operator
```

## Production note

The examples in this repository are designed to be easy to run locally.

For production usage:

- use a real migration image that runs your schema tool
- use a real restore image that understands your backup source
- keep database credentials in namespace-local Secrets
- change `migrationID` and `triggerNonce` intentionally to control reruns
