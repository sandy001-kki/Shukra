# Architecture

Shukra Operator is split into clear runtime layers so contributors can reason
about the platform without reading the whole codebase at once.

## Components

1. `api/`
   Defines the `AppEnvironment` schemas for `v1alpha1` and `v1beta1`, plus the
   explicit conversion logic used by the webhook conversion path.
2. `webhooks/`
   Applies defaults, validates policy and immutability rules, and delegates
   version conversion through the hub-and-spoke pattern.
3. `controllers/`
   Runs the ordered reconcile loop, classifies transient versus permanent
   failures, updates status, and emits events and metrics.
4. `internal/resources/`
   Contains deterministic resource builders for each managed child object.
5. `internal/finalizer/`
   Handles safe deletion and the mocked external cleanup calls.
6. `pkg/metrics` and `pkg/events`
   Centralize telemetry so operational behavior is consistent across the
   controller code path.

## Reconcile loop flow

1. Fetch `AppEnvironment`.
2. Short-circuit if it has already been deleted.
3. If deletion is in progress, run finalizer cleanup and remove the finalizer.
4. Ensure the finalizer exists.
5. Refresh observed generation and spec hash.
6. Honor paused mode without mutating child resources.
7. Validate dependency and policy requirements.
8. Reconcile resources in deterministic order:
   ConfigMap, secret references, Service, Deployment, HPA, migration Job,
   restore Job, Ingress, NetworkPolicy, PDB, backup CronJob.
9. Refresh conditions, phase, and last successful reconcile time.

This ordering keeps rollout behavior predictable and reduces partial-state
surprises during normal reconciliation and recovery.

## Webhook flow

1. The mutating webhook applies defaults such as replica count, service
   settings, probe defaults, and ingress path defaults.
2. The validating webhook rejects invalid specs before they reach the
   controller. This includes ingress host uniqueness, immutable field changes,
   and cross-namespace reference rejection.
3. The conversion webhook upgrades and downgrades between `v1alpha1` and
   `v1beta1`, including the lossy downgrade annotation.

## Leader election

Leader election is enabled through controller-runtime Leases in the operator
namespace. In a multi-replica deployment, one Pod actively reconciles while the
others stay hot and ready to take over if the leader exits or loses its lease.
This prevents duplicate writes while still allowing highly available rollouts.

## Tenancy model

Namespace is the tenant boundary for secrets, config, services, and workload
ownership. The only cluster-wide policy check is ingress hostname uniqueness,
which is enforced intentionally to avoid shared-L7 routing collisions between
tenants.
