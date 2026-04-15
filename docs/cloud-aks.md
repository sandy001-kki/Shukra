# Deploying Shukra on AKS

This guide highlights AKS-specific considerations for running Shukra.

## Main checks

- confirm `kubectl` points at the intended AKS cluster
- validate that network policy mode supports your app requirements
- ensure cert-manager and webhook services are reachable from the API server
- review any Azure policy constraints in the operator namespace

## Install flow

```bash
shukra doctor
helm install shukra-operator oci://ghcr.io/sandy001-kki/charts/shukra-operator \
  --version 0.2.3 \
  -n shukra-system \
  --create-namespace \
  -f charts/shukra-operator/values-production.yaml
```

## Notes

- confirm image pull policy and registry access if your cluster uses private registries
- align ingress, metrics, and cert-manager with your platform standards
- review namespace RBAC and security settings before multi-team adoption
