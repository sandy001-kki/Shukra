# GitOps with Flux

Flux can manage both the Shukra Helm chart and the `AppEnvironment` resources
that the operator reconciles.

Recommended pattern:

1. use a HelmRelease for the operator
2. store `AppEnvironment` manifests in the same or a related Git source
3. keep chart upgrades and tenant environment changes reviewable in pull requests

## Notes

- pin image and chart versions explicitly
- use `values-production.yaml` as a baseline and adapt it per environment
- keep Secret creation outside Shukra and reference the materialized Secret only
