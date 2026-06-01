# SPEC-120: SDD Spec Index Guard

## Status

Accepted for the current iteration.

## Context

`docs/sdd/specs/agent-gateway/000-index.md` is the navigation surface for the
Go migration SDD record. Recent specs can be added without updating the index,
which weakens the SDD process and makes architecture decisions harder to audit.

## Boundary Analysis

Go owns:

- runtime code and tests under `services/agent-runtime`.

Python owns:

- lightweight repository governance tests for SDD/document contracts.

Out of scope:

- runtime behavior changes;
- HTTP/MQ/API changes;
- AI worker behavior, model routing, RAG strategy, OCR/VLM, or image generation.

## Decision

Add a repository-level Python test that requires every
`docs/sdd/specs/agent-gateway/[0-9][0-9][0-9]-*.md` spec to be referenced by
exact filename in `000-index.md`.

Also update the index to include missing agent-gateway specs so the guard starts
green.

## Acceptance

- `000-index.md` references all current agent-gateway spec files.
- The new Python SDD index test passes.
- Existing contract fixture tests still pass.
- Go regression still passes.
- TODO is cleared at the end of the iteration.
