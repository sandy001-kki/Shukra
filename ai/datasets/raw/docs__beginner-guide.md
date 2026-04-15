# Beginner Guide

This guide is for someone who is completely new to Docker, Kubernetes, Helm,
and Operators.

If you are thinking "I do not know what these words mean, but I want to use
this project," this file is for you.

## The big picture

Shukra Operator helps you run an application on Kubernetes with one main YAML
file.

Without Shukra, you often need to create many Kubernetes files by hand:

- Deployment
- Service
- ConfigMap
- Ingress
- autoscaling configuration
- backup jobs
- migration jobs
- network security rules

With Shukra, you mostly create one custom resource:

- `AppEnvironment`

Shukra reads that one file and creates the lower-level Kubernetes resources for
you.

## What Docker is

Docker is a tool for packaging software into a container image.

You can think of a container image as a ready-to-run software package that
includes:

- the application
- the runtime
- required libraries
- the command used to start it

Why Docker matters here:

- the Shukra Operator itself runs as a container
- user applications usually run as containers too
- Kubernetes is built to run containers

So Docker is how software is packaged before Kubernetes runs it.

## What Kubernetes is

Kubernetes is a platform that runs and manages containers.

Instead of starting processes manually, Kubernetes handles things like:

- starting containers
- restarting them if they fail
- running multiple replicas
- exposing them on a network
- updating them safely
- attaching configuration and secrets

So Kubernetes is the system that runs applications in a reliable way.

## What kind is

`kind` means "Kubernetes in Docker."

It gives you a small local Kubernetes cluster on your own machine. This is very
useful for learning and development because you do not need a cloud cluster just
to test the operator.

Why it matters here:

- our local one-command bootstrap uses kind
- it lets you try Shukra safely on your laptop

## What Helm is

Helm is a package manager for Kubernetes.

It is similar to installing software from a package system, but for Kubernetes
applications.

Why it matters here:

- Helm installs the Shukra Operator into the cluster
- Helm also installs its related Kubernetes resources such as Deployment, RBAC,
  Services, and webhook configuration

## What an Operator is

A Kubernetes Operator is a controller that watches for a custom kind of object
and takes action automatically.

Shukra is an Operator.

That means:

- you create an `AppEnvironment`
- Shukra sees it
- Shukra creates the real Kubernetes resources needed
- Shukra keeps them updated over time
- Shukra cleans them up on deletion

So Shukra is not just a template generator. It is an always-running automation
system inside Kubernetes.

## What Shukra does

Shukra creates and manages a full application environment from one YAML.

Depending on the spec, it can manage:

- Deployment
- Service
- ConfigMap
- Ingress
- HorizontalPodAutoscaler
- migration Job
- restore Job
- backup CronJob
- NetworkPolicy
- PodDisruptionBudget

That is why the project says:

`One YAML. Complete Environment.`

## What the user actually writes

A user mainly writes one `AppEnvironment`.

Example:

```yaml
apiVersion: apps.shukra.io/v1beta1
kind: AppEnvironment
metadata:
  name: basic-app
spec:
  app:
    image: nginx:1.27
    replicas: 2
```

That is the high-level desired state.

Shukra then turns that into normal Kubernetes resources.

## What happens after the YAML is applied

When you run:

```bash
kubectl apply -f examples/basic.yaml
```

Shukra does work like this:

1. validate the resource
2. fill defaults where needed
3. ensure a finalizer exists
4. create or update child resources
5. write status and conditions back to the `AppEnvironment`

So the custom resource becomes the main source of truth.

## Why this is useful

For a beginner, Kubernetes has a steep learning curve because there are many
resource types.

Shukra reduces that learning burden.

Instead of first mastering every Kubernetes object, a user can begin with:

- what image to run
- how many replicas to use
- whether ingress is needed
- whether autoscaling, migration, backup, or restore are needed

This is the main value of the project.

## The easiest way to try Shukra

If you are on Windows, the repository includes a one-command bootstrap:

```powershell
powershell -ExecutionPolicy Bypass -File .\hack\bootstrap-local.ps1
```

That command will:

- start Docker Desktop
- create a local kind cluster
- install cert-manager
- build the Shukra operator image
- load the image into kind
- install the Shukra Helm chart
- apply the basic example

So you do not need to do each setup step manually.

## What to look at after bootstrap

Useful commands:

```bash
kubectl get appenvironment basic-app -n default -o yaml
kubectl get deploy,svc,cm,pods -n default
kubectl logs -n shukra-system deploy/shukra-operator
```

These commands show:

- the high-level custom resource
- the resources Shukra created
- the operator logs

## What status means

Shukra writes status back onto the `AppEnvironment`.

Important fields include:

- `status.phase`
- `status.conditions`
- `status.childResources`
- `status.lastError`
- `status.failureCount`

This means users can inspect one place first instead of manually hunting across
many Kubernetes objects.

## What happens when you delete the YAML

If you delete the `AppEnvironment`, Shukra runs finalizer logic before the
resource disappears.

That means cleanup happens safely instead of just removing the high-level
object and leaving everything else behind.

## The simple mental model

Use this model:

- Docker packages software
- Kubernetes runs and manages containers
- Helm installs Kubernetes software
- Shukra is an Operator running inside Kubernetes
- `AppEnvironment` is the one main file the user writes

If you understand those five points, you can start using this project.

## Where to go next

- `docs/getting-started.md` for the shortest hands-on workflow
- `README.md` for the overall project guide
- `docs/architecture.md` for the internal design
- `docs/tenancy.md` for namespace and secret rules
- `docs/troubleshooting.md` for common problems

