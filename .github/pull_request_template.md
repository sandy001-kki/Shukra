## Summary

Describe the change in a few short sentences.

## Why this change?

Explain the problem being solved and why this approach was chosen.

## What changed?

- 

## Validation

- [ ] `make generate`
- [ ] `make manifests`
- [ ] `go test ./...`
- [ ] `make lint`
- [ ] `make cli-build`
- [ ] `helm lint charts/shukra-operator`

Add any extra validation you ran:

```text
paste commands or results here
```

## API, tenancy, and security review

- [ ] This change does not weaken namespace tenancy
- [ ] This change does not log secret values
- [ ] This change preserves migration and restore idempotency
- [ ] This change does not introduce broad RBAC without justification
- [ ] This change preserves upgrade compatibility, or the impact is documented

## Docs

- [ ] README updated if user-facing behavior changed
- [ ] CLI docs updated if CLI behavior changed
- [ ] Learning docs updated if onboarding flow changed

## Release impact

- [ ] No release impact
- [ ] Release workflow changed
- [ ] Chart version or packaging changed
- [ ] CLI release assets changed
