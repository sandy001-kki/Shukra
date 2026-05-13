# Observability

Shukra exposes status, events, logs, and metrics so teams can understand what
the operator is doing.

## What to observe

- `AppEnvironment` phase and conditions
- operator logs
- Kubernetes events
- Prometheus metrics

## Metrics

Shukra defines metrics for:

- reconcile duration
- reconcile failures
- active environments
- migrations
- restores
- backup configuration
- AIONOS intent evaluation results
- active shadow environments
- shadow TTL expirations
- bridge patch applications
- active bridge stream connections

The AIONOS metrics are:

- `aionos_intent_evaluation_total`
- `aionos_shadow_environments_active`
- `aionos_shadow_environment_ttl_expirations_total`
- `aionos_bridge_patch_applications_total`
- `aionos_bridge_stream_connections_active`

## AIONOS events

The operator and bridge emit Kubernetes events for intent and bot activity:

- `AionosIntentViolated`
- `AionosIntentSatisfied`
- `AionosPatchApplied`
- `AionosShadowCreated`
- `AionosShadowExpired`

## Recommended workflow

1. start with `shukra doctor`
2. inspect `AppEnvironment` conditions
3. check operator logs
4. use your cluster monitoring stack for long-term trends

## Prometheus and Grafana

If your platform runs Prometheus, keep metrics enabled in Helm values and wire
the operator Service into your normal scrape configuration.

For Grafana, start with dashboards that track:

- reconcile failures over time
- active environments by namespace
- migration and restore counts
