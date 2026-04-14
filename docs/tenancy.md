# Namespace Tenancy

Shukra uses Kubernetes namespaces as the tenant isolation boundary.

## What is allowed inside one namespace

- Referencing existing `Secret` objects in the same namespace
- Referencing a namespace-local `ServiceAccount`
- Managing the owned Deployment, Service, ConfigMap, HPA, Ingress, Job,
  CronJob, NetworkPolicy, and PDB objects for one `AppEnvironment`

## What is forbidden across namespaces

- Secret references that imply `other-namespace/secret-name`
- Cross-namespace ConfigMap references
- Cross-namespace Service references
- Any operator-created workload wiring that escapes the owning namespace

These restrictions are deliberate. They keep the operator aligned with
namespace-scoped tenancy models used by most platform teams.

## Ingress host uniqueness

Ingress host uniqueness is enforced cluster-wide, even though namespaces are
tenant boundaries. This avoids two teams claiming the same hostname and
silently routing traffic to the wrong application.

## Secret strategy

- The operator never creates Secrets from inline values
- The operator only references existing Secrets
- The operator never logs secret values
- External Secrets Operator works by materializing a Secret first, then letting
  Shukra reference that Secret by name

## Watch scope

By default, the operator can watch all namespaces. To narrow that scope, set
`watchNamespace` in the Helm chart values. This is useful for smaller or more
isolated installations where one controller instance should only manage one
tenant namespace.
