# SDD Process

## Goal

Keep AI-assisted development controlled by explicit specifications. The spec is
the source of truth; code is accepted only when it matches the spec and preserves
the invariants.

## Workflow

0. **Iteration Contract**
   - At the start of a turn, read `TODO.md`, `DONE.md`, `LIVE_CHECKS.md`,
     `OPEN_ISSUES.md`, `BACKLOG.md`, and `git status`.
   - `TODO.md` is the current iteration contract. Finish every unchecked item
     in it before ending the iteration.
   - If an item cannot be finished because of an external blocker, record the
     blocker, required condition, and resume command. Otherwise do not leave
     partial TODO items behind.
   - Put unresolved risks, blockers, gaps, and decisions in `OPEN_ISSUES.md`;
     put future ideas in `BACKLOG.md` and live/manual validation in
     `LIVE_CHECKS.md`; do not inflate `TODO.md` with work that is not committed
     for the current iteration.

1. **Problem Statement**
   - Describe the user-visible problem.
   - State non-goals to prevent scope creep.

2. **Specification**
   - Define inputs, outputs, event schemas, state transitions, and invariants.
   - Define failure behavior and observability.
   - Define acceptance tests before implementation.

3. **Architecture Check**
   - Verify the change fits existing module boundaries.
   - If it creates a new boundary, write an ADR.
   - If a file would grow past 500 lines, split first.

4. **Implementation**
   - Implement the smallest complete vertical slice that satisfies the whole
     current `TODO.md` contract.
   - Keep domain logic pure where possible.
   - Put IO behind ports/adapters.

5. **SDD Review**
   - Review against the spec, not against intent guessed from code.
   - Check invariants, retries, idempotency, and failure handling.
   - Record review findings under `reviews/` when AI generated the code.

6. **Acceptance**
   - Run listed tests.
   - Record any gaps or deferred work.

## Review Gate

A change is not accepted if:

- it has no spec for new cross-module behavior;
- it introduces routing or safety logic inside adapters;
- it creates hidden coupling between Python and Go internals;
- it lacks tests for loop prevention, dedupe, or retry behavior;
- it expands an already oversized module instead of extracting a boundary.
- it ends with unchecked current-iteration TODO items that are not explicitly
  blocked by external state.
- it leaves unresolved risks or follow-up decisions outside `OPEN_ISSUES.md`.

## Go / Python Ownership Rule

- Go owns deterministic runtime infrastructure: routing state, idempotency,
  durable stores, lifecycle transitions, retries, leases, checkpoints, assets,
  queues, audit, and operational diagnostics.
- Python owns agent intelligence: model/provider routing, prompt and context
  pipelines, tool execution, memory/RAG extraction and ranking, embeddings,
  chunking, rerank, OCR/VLM, image-generation execution, group-knowledge
  distillation, evaluation scripts, and fast AI experiments.
- Python may keep compatibility mirrors while migrating, but mirrors must not
  become a second source of truth for infrastructure state once Go has the
  matching domain API.
- If a task touches both sides, Go owns the lifecycle and audit trail while
  Python owns the AI algorithm and provider behavior. The SDD spec must name
  both responsibilities explicitly.
