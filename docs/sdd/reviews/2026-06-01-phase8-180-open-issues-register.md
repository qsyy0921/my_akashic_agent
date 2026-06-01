# Phase 8 Review 180: Open Issues Register

## Scope

- Add `docs/sdd/OPEN_ISSUES.md` as the canonical unresolved issue ledger.
- Document how it differs from `TODO.md`, `DONE.md`, `BACKLOG.md`, and `LIVE_CHECKS.md`.
- Add a lightweight governance test so the document remains present and role-aware.

## Design Check

- `TODO.md` remains a short current-iteration commitment list and is cleared before the end of the slice.
- `OPEN_ISSUES.md` contains unresolved issues, risks, pending decisions, and known gaps; those are not automatically current-iteration tasks.
- Backlog remains for future candidate work and planning, while live checks remain runtime/manual verification.

## Verification

- `uv run pytest tests/test_sdd_governance_docs.py -q`
- `uv run pytest tests/test_sdd_spec_index.py -q`
- `git diff --check`

## Risk

- The issue ledger will only stay useful if future iterations update it when discovering new gaps. The test enforces existence and roles, not content quality.

## Result

Accepted. Unresolved issues now have a dedicated SDD register separate from TODO and DONE.
