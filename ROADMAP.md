# Roadmap

This file tracks the most useful next milestones for Shukra Operator.

The roadmap is intentionally practical: each item should improve usability,
operability, or production readiness for real Kubernetes users.

## Near term

- Add richer CLI commands for logs, events, and rollout inspection
- Improve migration and restore status tracking with richer Job condition handling
- Add a `SECURITY.md` driven disclosure process to the release notes and docs
- Add more CI checks for webhook conversion and upgrade flows
- Improve conflict retry handling in reconciliation to reduce transient update churn
- Add release badges that surface the latest public image, chart, and CLI assets

## Short term

- Add namespace-scoped watch mode examples and Helm presets
- Add ServiceMonitor and dashboard examples for Prometheus/Grafana users
- Add more complete restore verification and post-restore health checks
- Add operator config docs for tuning concurrency, defaults, and watch namespace
- Add platform-focused example stacks for web API, worker, and internal service patterns
- Add release notes automation with changelog generation

## Medium term

- Introduce a `v1` API graduation plan with deprecation guidance
- Add pluggable external integrations for backup and DNS cleanup clients
- Add richer policy controls for ingress ownership and shared gateways
- Support controlled rollout strategies for migrations and restore orchestration
- Add stronger status summaries for backups, restores, and autoscaling health
- Add multi-cluster install docs and operational patterns

## Longer term

- Optional UI/dashboard for AppEnvironment visibility
- GitOps starter packs for Argo CD and Flux
- Policy packs for common compliance baselines
- Advanced multi-team tenancy presets
- Extended CLI workflows for day-two SRE operations

## How to use this roadmap

- Treat it as directional, not guaranteed by date
- Open issues for roadmap items before implementation
- Link pull requests back to the roadmap item they advance
- Propose additions when they improve user value or production safety
