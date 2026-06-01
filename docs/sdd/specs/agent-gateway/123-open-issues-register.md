# SPEC-123: Open Issues Register

## Status

Accepted for the current iteration.

## Context

`TODO.md`, `DONE.md`, `BACKLOG.md`, and `LIVE_CHECKS.md` already exist, but they
do not provide a single issue ledger for unresolved risks, open decisions, and
known gaps. As the Go migration grows, mixing open issues into TODO or BACKLOG
makes each iteration harder to close cleanly.

## Boundary Analysis

Go owns:

- deterministic runtime implementation and architecture guardrails.

Python owns:

- AI execution and repository-level governance tests when needed.

SDD owns:

- a clear unresolved-issue register that feeds future TODO slices.

Out of scope:

- solving every open issue in this slice;
- changing runtime behavior;
- changing dashboard UI, MQ cutover, media downloader, RAG strategy, or AI
  worker execution.

## Decision

Add `docs/sdd/OPEN_ISSUES.md` as the canonical unresolved issue ledger.

Document the split:

- `TODO.md`: current iteration tasks only;
- `DONE.md`: completed facts;
- `BACKLOG.md`: future candidates and planning;
- `LIVE_CHECKS.md`: manual/live verification;
- `OPEN_ISSUES.md`: unresolved issues, risks, decisions, and gaps.

## Acceptance

- `OPEN_ISSUES.md` exists and lists current known unresolved issues.
- `TODO.md` is still cleared at the end of the iteration.
- SDD review/DONE/LIVE_CHECKS are updated.
