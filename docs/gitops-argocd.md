# GitOps with Argo CD

Shukra fits well with GitOps because the operator remains declarative.

Recommended pattern:

1. store the Shukra Helm release in Git
2. store `AppEnvironment` manifests in Git
3. let Argo CD apply both
4. let the operator reconcile child resources

## Typical split

- one Argo CD application for the operator chart
- one or more Argo CD applications for tenant `AppEnvironment` resources

## Notes

- pin chart versions explicitly
- avoid editing live resources outside Git except for emergency troubleshooting
- keep Secrets externalized and reference only materialized same-namespace Secrets
