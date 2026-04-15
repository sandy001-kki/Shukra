# Shukra AI Architecture

This document explains the technical design for adding real AI capabilities to
Shukra without weakening safety, correctness, or the operator’s declarative
model.

## Design principles

- the operator remains the source of truth for cluster behavior
- AI assists users, but does not replace deterministic reconciliation
- knowledge must be grounded in the Shukra repo, CRD, and docs
- unsafe actions must never be executed implicitly
- secret values must never be shown or inferred from logs

## System layers

```text
User
  -> Shukra CLI / chat
  -> Intent interpretation layer
  -> Grounding layer (docs, examples, CRD, status)
  -> Optional model runtime
  -> Guardrail layer
  -> Deterministic execution layer
  -> Kubernetes / Shukra Operator
```

## Layer 1: User interface

Primary entrypoints:

- `shukra chat`
- future `shukra ask`
- future `shukra generate`
- future `shukra diagnose`

The interface should accept plain English while keeping output easy to trust and
easy to verify.

## Layer 2: Intent interpretation

This layer decides what the user is asking for:

- explanation
- generation
- diagnosis
- install help
- status summary
- mutation request

Today, the repo already includes a deterministic English parser for common chat
commands. That is the first bridge toward a fuller AI interface.

## Layer 3: Grounding

Before any answer is produced, the system should ground itself in:

- docs
- examples
- API types
- CRD schema
- current status from the cluster, when relevant

Grounding data should be chunked and indexed so responses remain tied to actual
project truth.

## Layer 4: Optional model runtime

This is the only layer that makes the system truly AI in the generative sense.

Possible deployment modes:

- local model runtime
- hosted internal model
- hosted third-party model

Without this layer, Shukra can still be retrieval-driven and assistant-like, but
it is not a real generative AI system.

## Layer 5: Guardrails

Every model or retrieval result must pass through guardrails that enforce:

- no cross-namespace secret or service references
- no unsupported CRD fields
- no unsafe shell suggestions by default
- no mutation without explicit user intent
- no secret output
- no claims that contradict the current API

## Layer 6: Deterministic execution

All real actions should happen through deterministic code:

- generate YAML from known templates or typed structs
- apply manifests via `kubectl`
- inspect `AppEnvironment` status via typed clients
- invoke install/bootstrap flows through the existing CLI code

This keeps the model advisory and the execution layer reliable.

## Retrieval-first recommendation

The recommended architecture is retrieval-first:

1. index Shukra knowledge
2. retrieve relevant chunks for each question
3. answer from those chunks
4. only later add fine-tuning for better style and domain fluency

This order keeps correctness higher than jumping directly to model tuning.

## Fine-tuning recommendation

If a model is added, fine-tune only for:

- domain vocabulary
- intent mapping
- YAML generation style
- troubleshooting explanation quality

Do not fine-tune the model to become the execution engine.

## Safety boundaries

The AI layer must never:

- create inline secret material
- print secret values
- bypass namespace tenancy rules
- invent status that is not present
- bypass the operator’s validation model

## Evaluation

Every assistant version should be evaluated on:

- beginner questions
- CLI guidance questions
- YAML generation tasks
- migration and restore scenarios
- safety refusal cases
- status interpretation tasks

## What the repo includes now

This repo now includes the planning and data-prep foundation for this
architecture:

- AI roadmap
- AI workspace structure
- dataset generation script
- readiness evaluation script

That allows the project to move into retrieval and model work later without
starting from scratch.
