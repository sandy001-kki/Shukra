# Learning Path

This file gives new users a clear sequence to learn Shukra without trying to
understand everything at once.

## Stage 1: Absolute beginner

Goal:

- understand Docker
- understand Kubernetes at a basic level
- understand what an Operator is
- understand what Shukra is doing at a high level

Read first:

- `docs/beginner-guide.md`

After this stage, you should be able to answer:

- Why does this project use Docker?
- Why does this project use Kubernetes?
- What does Shukra automate?
- What is an `AppEnvironment`?

## Stage 2: First hands-on success

Goal:

- run Shukra locally
- install the operator
- apply one working environment
- inspect the result

Do this:

1. Run the one-command bootstrap:

   `powershell -ExecutionPolicy Bypass -File .\hack\bootstrap-local.ps1`

2. Inspect the cluster:

   - `kubectl get appenvironment basic-app -n default -o yaml`
   - `kubectl get deploy,svc,cm,pods -n default`
   - `kubectl logs -n shukra-system deploy/shukra-operator`

Read:

- `docs/getting-started.md`

After this stage, you should be able to answer:

- What resources did Shukra create?
- Where do I look for status?
- How do I know the sample app is running?

## Stage 3: Understand the user-facing API

Goal:

- understand what fields a normal user edits
- understand what those fields cause Shukra to do

Read:

- `README.md`
- `docs/api.md`

Focus on:

- `spec.app`
- `spec.service`
- `spec.ingress`
- `spec.autoscaling`
- `spec.database`
- `spec.migration`
- `spec.restore`
- `spec.backup`
- `spec.security`

After this stage, you should be able to create your own `AppEnvironment`
instead of copying the examples unchanged.

## Stage 4: Run richer examples

Goal:

- try common platform features one by one
- start using the CLI for common lifecycle actions

Recommended order:

1. `examples/basic.yaml`
2. `examples/paused.yaml`
3. `examples/autoscaling.yaml`
4. `examples/ingress.yaml`
5. `examples/migration.yaml`
6. `examples/restore.yaml`

Why this order:

- `basic` proves the core path works
- `paused` shows lifecycle control
- `autoscaling` adds HPA behavior
- `ingress` adds external routing
- `migration` adds database workflow
- `restore` adds recovery workflow

Helpful CLI commands during this stage:

- `shukra env status <name> -n default`
- `shukra env pause <name> -n default`
- `shukra env resume <name> -n default`
- `shukra env migrate <name> -n default --migration-id ...`
- `shukra env restore <name> -n default --trigger-nonce ... --image ... --source ...`

## Stage 5: Understand production rules

Goal:

- learn the constraints Shukra enforces for safety

Read:

- `docs/tenancy.md`
- `docs/troubleshooting.md`

Important ideas:

- namespace is the tenant boundary
- secrets are reference-only
- ingress hosts are globally unique
- finalizers protect delete flow
- migration and restore use idempotency keys

## Stage 6: Understand internal architecture

Goal:

- understand how the operator is built internally
- understand reconcile flow, webhooks, and status handling

Read:

- `docs/architecture.md`

After this stage, you should understand:

- why reconciliation is ordered
- how webhooks and conversion fit in
- why leader election matters
- why child resources are builder-driven

## Stage 7: Contributor path

Goal:

- contribute code safely
- run generation and tests
- understand release workflow

Read:

- `CONTRIBUTING.md`
- `docs/cli.md`
- `.github/workflows/ci.yaml`
- `.github/workflows/release.yaml`

Run:

- `make generate`
- `make manifests`
- `make test`
- `make lint`

## Suggested path in one line

`beginner-guide -> getting-started -> README/API -> examples -> tenancy/troubleshooting -> architecture -> contributing`

