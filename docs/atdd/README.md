# ATDD Guide

## Goal

ATDD means `Acceptance Test-Driven Development`.
It defines what "done" means from the operator, user, or system integration
point of view before or alongside implementation.

In this repository:

- `SDD` defines the architecture, boundaries, invariants, and side effects.
- `ATDD` defines the acceptance scenarios that must pass before a slice is
  considered complete.
- `TDD` defines the lower-level automated tests that protect the behavior.

ATDD is the bridge between spec and runtime proof.

## When To Write ATDD

Write or update ATDD whenever a slice changes any of these:

- operator-visible workflow;
- runtime endpoint contract or dashboard behavior;
- queue, job, media, scheduler, receiver, or delivery state transitions;
- manual smoke path that determines whether cutover is safe;
- any behavior that needs a human or integration-level confirmation.

Pure refactors that do not change observable behavior usually do not need a new
ATDD document, but must still satisfy existing ATDD coverage.

## File Layout

Recommended pattern:

- one ATDD file per slice or per capability;
- place it near the relevant area under `docs/atdd/`;
- keep names stable and descriptive.

Examples:

- `docs/atdd/qq-live-send-smoke.md`
- `docs/atdd/media-content-recovery.md`
- `docs/atdd/knowledge-planner-cutover.md`

## Required Sections

Each ATDD document should include:

1. Scope
2. Preconditions
3. Scenarios
4. Expected Results
5. Failure Signals
6. Evidence

## Scenario Template

Use this template:

```md
# ATDD: <capability name>

## Scope

- What user-visible or operator-visible behavior this covers.

## Preconditions

- Required services, env vars, accounts, fixtures, or data.

## Scenarios

### Scenario 1

- Action:
  Perform the concrete workflow.
- Expect:
  Describe the visible result, persisted state, and logs/metrics if relevant.

### Scenario 2

- Action:
  Perform the failure or edge-case workflow.
- Expect:
  Describe the safe failure behavior and what must not happen.

## Failure Signals

- Exact signs that mean the acceptance test failed.

## Evidence

- Commands, screenshots, URLs, logs, or artifacts to retain.
```

## Relation To LIVE_CHECKS

- `docs/sdd/LIVE_CHECKS.md` is the rolling checklist of concrete live checks.
- `docs/atdd/*.md` is the structured acceptance contract behind those checks.

If a live check is important enough to decide whether a slice is complete or a
cutover is safe, it should have an ATDD document.
