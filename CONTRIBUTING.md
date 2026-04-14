# Contributing

This file explains the contribution expectations for Shukra Operator.

Shukra is a production-oriented Kubernetes Operator, so changes need to protect
API compatibility, safety, and day-two operations, not just code style.

## Contribution workflow

1. Fork the repository at [github.com/sandy001-kki/Shukra](https://github.com/sandy001-kki/Shukra).
2. Create a focused branch from `main`.
3. Make your change with clear commit messages.
4. Run the required generation and validation steps before opening a pull request.
5. Open the pull request with a clear description of:
   - what changed
   - why it changed
   - any API or behavior impact
   - how you tested it

## Required checks before a pull request

Run these commands locally unless your change is documentation-only:

```bash
make generate
make manifests
go test ./...
make lint
make cli-build
helm lint charts/shukra-operator
```

If you changed release or install behavior, also validate:

```bash
make docker-build
```

If you changed local bootstrap behavior on Windows, validate:

```powershell
powershell -ExecutionPolicy Bypass -File .\hack\bootstrap-local.ps1
```

## Special requirements

The following rules are especially important in this repository:

- Do not break CRD compatibility casually. API changes must consider upgrade and conversion behavior.
- Do not remove or weaken namespace tenancy protections.
- Do not add inline secret creation flows. Shukra is reference-only for secrets.
- Do not log secret values at any level.
- Do not bypass finalizers for delete paths unless the change is explicitly justified and tested.
- Do not introduce non-idempotent reconciliation behavior.
- Do not change migration or restore behavior without preserving idempotency semantics.
- Do not add broad RBAC without a least-privilege justification.
- Keep user-facing docs updated when commands, examples, install flow, or releases change.
- Keep the CLI and operator docs aligned. If one changes, the other usually needs an update too.

## Code review expectations

Reviewers will look closely at:

- API compatibility
- reconciliation order and idempotency
- safety of delete and cleanup paths
- tenancy isolation
- secret handling
- observability changes
- Helm/install correctness
- release workflow correctness
- beginner and operator-facing documentation quality

## Release-related changes

If your pull request affects any of the following, call it out explicitly:

- `.github/workflows/release.yaml`
- `charts/shukra-operator/`
- `Dockerfile`
- `cmd/shukra/`
- `pkg/cli/`
- CRD or webhook behavior

These changes have downstream impact on published images, OCI charts, and CLI
release binaries.

## Documentation responsibility

Update the relevant docs when you change behavior:

- `README.md` for user-facing install and usage changes
- `docs/cli.md` for CLI changes
- `docs/getting-started.md` for beginner workflow changes
- `docs/learning-path.md` if the learning sequence changes
- `docs/migration-restore-walkthrough.md` for migration/restore behavior changes
- `docs/tenancy.md` for namespace or secret model changes
- `docs/troubleshooting.md` for operator failure mode changes

## Pull request quality bar

A strong pull request in this repository is:

- small enough to review clearly
- tested locally
- documented
- safe for upgrades
- safe for multi-tenant clusters
- explicit about operational impact
