# Deploying Shukra on GKE

This guide highlights GKE-specific considerations for running Shukra.

## Main checks

- confirm `kubectl` points at the intended GKE cluster
- make sure webhook traffic is allowed by cluster policy
- review ingress class settings for your preferred ingress controller
- confirm cert-manager is permitted to create required resources

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

- validate any organization policy around Pod security and admission controllers
- integrate metrics with your Prometheus or managed monitoring setup
- choose an issuer strategy that matches your certificate governance model
