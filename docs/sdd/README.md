# SDD Development Record

This directory records the Specification-Driven Development process used for
Akashic Memory Agent changes, especially the planned Go agent runtime and
multi-channel messaging refactor.

SDD here means: write the behavior contract first, implement against that
contract, then review code by checking the contract, invariants, and tests.
For the Go migration, each iteration treats `TODO.md` as a closure contract:
all unchecked current-iteration items must be completed, moved back to
`BACKLOG.md`, or marked with a real external blocker before the iteration ends.

## Directory Layout

- `PROCESS.md`: the workflow for spec, implementation, review, and acceptance.
- `ITERATION_PROMPT.md`: resume prompt for future Codex sessions, including
  TODO closure and Go/Python ownership rules.
- `templates/spec-template.md`: required structure for a feature spec.
- `templates/review-checklist.md`: checklist used for AI-generated code review.
- `IMPLEMENTATION_FREEZE.md`: active gate for Go/Python split and group memory
  migration work.
- `adr/`: architecture decision records.
- `specs/agent-gateway/`: specs for the Go agent runtime (named `agent-runtime` in
  service layout).
- `specs/agent-architecture/`: specs for Python/Go ownership and target agent
  architecture.
- `specs/group-message-memory/`: specs for QQ/group message memory, RAG, and
  evaluation.
- `reviews/`: per-change review records.

## Required Artifacts For Each Major Change

Every non-trivial feature should include:

1. A spec under `specs/<area>/NNN-name.md`.
2. Tests linked from the spec acceptance section.
3. A review note under `reviews/` when AI-generated code is involved.
4. An ADR when the change creates or changes an architectural boundary.

## Current Focus

The current high-risk area is messaging:

- multiple QQ accounts;
- bot-to-bot interaction;
- provenance classification;
- loop prevention;
- queued media/file/memory processing;
- Go runtime boundary design.
- group-message memory and RAG.

For the Go/Python architecture split, group-message memory, and RAG migration,
`IMPLEMENTATION_FREEZE.md` is active as a cutover gate. Non-production Go
control-plane slices may be implemented with review records and tests; production
platform cutover still requires a specific review and rollback plan.

## Ownership Summary

- Go owns deterministic backend runtime: account-aware routing, durable inbox /
  outbox, leases, retries, dedupe, queue lifecycle, media assets, checkpoints,
  scheduling control plane, audit, and dashboard diagnostics.
- Python owns AI runtime behavior: model/provider selection, prompt/context
  construction, tool execution, memory/RAG algorithms, chunking,
  embedding/rerank, retrieval strategy, OCR/VLM enrichment, image-generation
  execution, group-knowledge distillation, evaluation, and fast
  provider-specific experiments.
