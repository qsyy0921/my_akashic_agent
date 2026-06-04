# TDD Guide

## Goal

TDD means `Test-Driven Development`.
It defines the automated test surface that protects the behavior introduced by a
slice.

In this repository:

- `SDD` defines architecture and behavior contracts.
- `ATDD` defines acceptance scenarios.
- `TDD` defines the automated tests that should fail before implementation and
  pass after implementation.

TDD is the regression guardrail.

## When To Write TDD

Write or update TDD whenever a slice changes:

- domain rules or state transitions;
- HTTP request/response contracts;
- idempotency, lease, retry, dedupe, or loop-prevention logic;
- dashboard normalization logic;
- boundary behavior between Go runtime and Python runtime.

## File Layout

Recommended pattern:

- one TDD file per slice or capability;
- place it under `docs/tdd/`;
- keep it close to the test intent rather than mirroring source tree layout.

Examples:

- `docs/tdd/outbound-cutover-readiness.md`
- `docs/tdd/receiver-lease-cleanup.md`
- `docs/tdd/media-content-recovery.md`

## Required Sections

Each TDD document should include:

1. Scope
2. Target Code Paths
3. Test Matrix
4. Required Automated Tests
5. Deferred Coverage

## Template

```md
# TDD: <capability name>

## Scope

- What behavior this test design protects.

## Target Code Paths

- Go packages, Python modules, dashboard code, or contract fixtures involved.

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| happy path | unit/integration/contract | key expected result |
| edge case | unit/integration/contract | safe edge behavior |
| failure path | unit/integration/contract | safe failure behavior |

## Required Automated Tests

- Exact `go test`, `pytest`, contract fixture, or build/typecheck commands.

## Deferred Coverage

- What is intentionally left to ATDD/live smoke, and why.
```

## Rules

- Prefer small deterministic tests first.
- Add contract tests when Go/Python or backend/dashboard payloads cross a
  boundary.
- Put runtime-only checks in ATDD or `LIVE_CHECKS.md`, not in TDD.
- Do not treat a slice as complete if the implementation changed but its test
  surface was not reviewed.
