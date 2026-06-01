# Phase 8 Review 181: Open Issues Registry Guard

## Scope

- Strengthen `tests/test_sdd_governance_docs.py`.
- Require `OPEN_ISSUES.md` rows to be structured and parseable.
- Keep the slice governance-only; no runtime, API, MQ, media, dashboard, RAG, or AI behavior changes.

## Design Check

- Open issue rows now require stable `OI-###` ids, non-empty area/problem/impact/next-step fields, unique ids, and a known status.
- The allowed statuses are `Open`, `In Progress`, `Blocked`, and `Resolved`.
- This keeps unresolved issues actionable without turning them into current-iteration TODO items.

## Verification

- `uv run pytest tests/test_sdd_governance_docs.py -q`
- `uv run pytest tests/test_sdd_spec_index.py -q`
- `go test ./...` from `services/agent-runtime`
- `git diff --check`

## Risk

- The guard validates structure, not whether every real-world issue has been captured. Review discipline is still required when discovering new risks.

## Result

Accepted. The unresolved issue ledger now has a machine-checked minimum structure.
