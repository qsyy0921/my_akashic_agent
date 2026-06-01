# SPEC-124: Open Issues Registry Guard

## Status

Accepted for the current iteration.

## Context

`OPEN_ISSUES.md` now records unresolved risks and gaps, but without automated
shape checks it can drift into free-form notes. That would make it harder to
promote issues into `TODO.md`, track status, and avoid losing architectural
decisions during the Go migration.

## Boundary Analysis

Go owns:

- runtime code and deterministic infrastructure.

Python owns:

- repository-level SDD governance tests.

SDD owns:

- the unresolved issue ledger and its minimum structure.

Out of scope:

- solving the listed open issues in this slice;
- changing runtime behavior, MQ, media downloading, RAG strategy, dashboard UI,
  or AI workers.

## Decision

Extend the SDD governance test so `OPEN_ISSUES.md` must keep a parseable table
with these columns:

- `ID`
- `领域`
- `问题`
- `影响`
- `下一步`
- `状态`

Each issue must have a stable `OI-###` id, non-empty fields, and a known status
value: `Open`, `In Progress`, `Blocked`, or `Resolved`.

## Acceptance

- The governance test fails on malformed open issue rows.
- Current `OPEN_ISSUES.md` passes.
- TODO is cleared at the end of the iteration.
