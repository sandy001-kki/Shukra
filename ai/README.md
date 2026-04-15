# Shukra AI Workspace

This directory exists to hold the artifacts needed to evolve Shukra from a
documentation-backed assistant into a model-backed assistant.

## Layout

- `datasets/raw/`
  extracted source material from repo docs and examples
- `datasets/generated/`
  generated JSONL datasets for Q&A, instructions, and tasks
- `eval/`
  evaluation prompts, checks, and scoring notes
- `prompts/`
  system and task prompt templates for future retrieval or model runs

## Current scope

The repository currently prepares the data and evaluation foundation.

That means:

- knowledge can be extracted from repo sources
- training-ready JSONL can be generated
- readiness checks can confirm the AI workspace is populated

It does not mean:

- a trained model is bundled with the repo
- a no-runtime generative AI system exists

## Commands

Generate the Shukra AI dataset:

```powershell
make ai-dataset
```

Run AI workspace readiness checks:

```powershell
make ai-eval
```

## Important constraint

To behave like a true AI assistant, Shukra still needs a model runtime
somewhere. This workspace prepares everything around that requirement, but it
does not eliminate it.
