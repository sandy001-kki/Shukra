# Shukra AI Roadmap

This document explains how to turn Shukra from a production-grade operator with
a friendly CLI into a genuinely AI-assisted platform workflow.

## Reality first

Two statements can both be true:

1. We can build the entire training, tuning, and evaluation pipeline for a
   Shukra-specific assistant.
2. We cannot make that assistant answer real questions without a model runtime.

So the roadmap is designed in phases that create value early and keep the
system grounded in the actual Shukra codebase.

## Goal

Build a Shukra-specific assistant that can:

- explain Shukra to new users
- generate safe `AppEnvironment` manifests
- diagnose install and reconcile issues
- suggest or run Shukra CLI workflows
- summarize status from the cluster
- stay aligned with the CRD, docs, and actual controller behavior

## Phase 0: Repo-grounded knowledge base

Objective:
- turn Shukra docs, examples, and CLI semantics into structured knowledge

Deliverables:
- corpus manifest of repo sources
- normalized text chunks from docs and examples
- Q&A and instruction datasets in JSONL
- readiness checks for required knowledge sources

What works at the end of this phase:
- documentation-backed answers
- dataset generation
- evaluation of knowledge coverage

What does not exist yet:
- true AI inference

## Phase 1: Retrieval assistant

Objective:
- answer Shukra questions by retrieving relevant repo knowledge before replying

Core behaviors:
- answer “what does this field do?”
- answer “how do I install Shukra?”
- answer “why did migration not run?”
- point users to docs and examples

Guardrails:
- answer only from indexed Shukra sources
- prefer “I don’t know” to guessing
- surface doc paths and examples as evidence

## Phase 2: Instruction dataset and synthetic tuning set

Objective:
- create a Shukra-only supervised fine-tuning dataset

Dataset categories:
- beginner Q&A
- YAML generation examples
- install and troubleshooting dialogues
- CLI help transformations
- status interpretation examples
- migration and restore idempotency explanations
- tenancy and security policy questions

Required properties:
- grounded in repo truth
- no invented features
- explicit safe/unsafe command boundaries

## Phase 3: Fine-tuned small model

Objective:
- adapt a small open model to Shukra’s language and task shapes

Recommended use cases:
- manifest drafting
- operator concept explanation
- CLI task interpretation
- troubleshooting summaries

Not recommended as the only source of truth:
- direct cluster mutation without guardrails
- free-form Kubernetes administration outside Shukra’s domain

## Phase 4: AI-assisted CLI

Objective:
- extend `shukra chat` from rule-based English parsing into a model-guided
  assistant while keeping execution safe

Suggested command families:
- `shukra ask`
- `shukra chat`
- `shukra explain`
- `shukra generate`
- `shukra diagnose`

Safety model:
- model proposes intent
- deterministic layer validates intent
- execution layer applies allowed actions only
- user-visible summaries explain what will happen

## Phase 5: Cluster-aware assistant

Objective:
- let the assistant combine repo knowledge with live cluster reads

Examples:
- summarize an `AppEnvironment`
- compare desired spec with current status
- diagnose why `Ready=False`
- explain `failureCount`
- show what child resources exist

Required constraints:
- read-only by default
- write actions always explicit
- no secret values shown
- no cross-namespace leakage

## Data sources

The Shukra assistant should learn from:

- `README.md`
- `docs/*.md`
- `examples/*.yaml`
- `api/` types
- CRD schemas
- CLI help text
- selected controller/status logic summaries

## Success criteria

The AI program is working well when it can:

- answer beginner questions in simple English
- generate valid starter manifests
- map user intent to safe Shukra workflows
- explain migration and restore behavior correctly
- refuse unsupported or unsafe requests clearly

## Non-goals

These are intentionally out of scope for the first versions:

- general-purpose Kubernetes assistant for every cluster task
- autonomous mutation of arbitrary resources
- secret inspection or secret generation
- replacing controller logic with model output

## Recommended order of work

1. Finish knowledge extraction and dataset generation
2. Add automated dataset quality checks
3. Build retrieval-backed answers
4. Fine-tune a small Shukra-domain model
5. Add model-guided chat with deterministic guardrails

## Current repository support

This repo already includes:

- `make ai-dataset`
- `make ai-eval`
- `hack/prepare-ai-dataset.ps1`
- `hack/evaluate-ai-readiness.ps1`
- `ai/README.md`

That means the planning and scaffolding layer is ready even before a model
runtime is chosen.
