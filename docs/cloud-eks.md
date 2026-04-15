# Deploying Shukra on EKS

This guide highlights EKS-specific considerations for running Shukra.

## Main checks

- confirm the current `kubectl` context points at the EKS cluster
- install cert-manager or use an approved certificate-management path
- make sure your ingress class and load balancer strategy match your platform
- review network policy support for your chosen CNI mode

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

- review IAM, IRSA, and registry pull behavior if you mirror images internally
- expose metrics using your cluster monitoring standards
- choose ingress annotations and class settings appropriate for ALB or NGINX
