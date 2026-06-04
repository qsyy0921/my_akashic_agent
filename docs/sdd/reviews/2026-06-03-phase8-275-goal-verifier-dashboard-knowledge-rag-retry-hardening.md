# Review: Goal Verifier Dashboard Knowledge/RAG Retry Hardening

## Scope

- `scripts/verify-go-migration-goal.ps1`
- `tests/test_verify_go_migration_goal_script.py`

## Findings

- The regression was in unified evidence sampling, not in the Go runtime or
  dashboard read-model itself.
- A single bounded retry on the dashboard knowledge/RAG verifier is sufficient
  because the standalone verifier already returns `live_verified` on the same
  turn.
- The change stays read-only and does not widen any runtime side effects.

## Verification

- `uv run pytest tests/test_verify_go_migration_goal_script.py tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-dashboard-knowledge-rag-state-boundary.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`

## Residual Risk

- The retry hardening only covers this one verifier path. Future dashboard
  verifier regressions on other read-models still need their own stability work
  if they show similar current-turn sampling drift.
