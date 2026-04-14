# Security Policy

Shukra Operator is designed for production Kubernetes environments, so security
reports are treated seriously and handled with care.

## Supported versions

The latest tagged release on `main` receives security fixes first.

| Version line | Supported |
| --- | --- |
| latest stable release | yes |
| older release lines | best effort only |
| unreleased forks | no |

If you are unsure whether your version is still supported, open a discussion or
upgrade to the latest release before filing a report.

## Reporting a vulnerability

Please do not open public GitHub issues for suspected security vulnerabilities.

Instead, report them privately through one of these channels:

1. GitHub Security Advisories for this repository, if enabled in the repo UI.
2. A private email to the maintainer address you publish for the project.

When reporting, include:

- affected Shukra version
- Kubernetes version
- whether the issue is cluster-local or publicly reachable
- exact steps to reproduce
- proof-of-concept YAML or commands if available
- any known mitigations

## What to expect

Maintainers should aim to:

1. acknowledge the report within 3 business days
2. confirm severity and impact as quickly as possible
3. prepare a fix or mitigation
4. publish a coordinated security release when ready

## Scope guidance

Examples of issues that are in scope:

- secret leakage in logs or status
- namespace tenancy bypass
- unsafe cross-namespace references
- webhook validation bypass that allows privilege escalation
- RBAC overreach that meaningfully expands cluster access
- restore, migration, or backup behavior that can be abused across tenants

Examples that are usually out of scope:

- vulnerabilities in third-party images you choose to run in your own examples
- insecure clusters that disable admission, RBAC, or other normal protections
- local development-only shortcuts used outside recommended production settings

## Hardening reminders

When deploying Shukra in production:

- keep the operator and CRDs on the latest release
- review RBAC before installation
- keep cert-manager and Kubernetes up to date
- avoid broad cluster access outside the operator namespace
- use existing secret managers or External Secrets Operator for secret delivery
- restrict who can create or update `AppEnvironment` resources
