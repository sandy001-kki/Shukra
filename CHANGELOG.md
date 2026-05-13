# Changelog

All notable changes to Shukra Operator should be documented in this file.

## v0.3.0

- added the AIONOS Bridge gRPC server as a separate `shukra-bridge` binary
- added intent declarations to `AppEnvironment` spec and intent health in status
- added the AIONOS shadow namespace protocol with TTL-based cleanup
- replaced mock deletion cleanup with pluggable database, backup, and DNS hooks
- hardened status updates and reconciliation paths for conflict retry
- added AIONOS Prometheus metrics and Kubernetes events
- added patch audit history in `AppEnvironment` status
- added Helm resources for the bridge Deployment, Service, certificate, and shadow namespace

## v0.2.3

- added `shukra doctor`
- added explicit `diagnose` commands
- expanded `shukra chat`
- improved bring-your-own-cluster documentation
- added production Helm values guidance

## v0.2.2

- polished README landing page
- improved CLI UX and control flows

## v0.2.1

- added learning path and richer release docs

## v0.2.0

- added beginner onboarding improvements

## v0.1.0

- initial public production-style Shukra Operator release
