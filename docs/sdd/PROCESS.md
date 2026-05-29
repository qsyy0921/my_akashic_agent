# SDD Process

## Goal

Keep AI-assisted development controlled by explicit specifications. The spec is
the source of truth; code is accepted only when it matches the spec and preserves
the invariants.

## Workflow

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
   - Implement the smallest vertical slice.
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

