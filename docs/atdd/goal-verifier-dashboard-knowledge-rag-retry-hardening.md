# ATDD: Goal Verifier Dashboard Knowledge/RAG Retry Hardening

## Scope

- Unified goal verifier current-turn artifact stability when the dashboard
  knowledge/RAG live boundary verifier briefly returns a non-live result.

## Preconditions

- Live Go runtime at `http://127.0.0.1:8780`
- Live dashboard at `http://127.0.0.1:2236`
- Repo-owned verifier
  `scripts/verify-dashboard-knowledge-rag-state-boundary.ps1`

## Scenarios

### Scenario 1

- Action:
  Run `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
  on the current live runtime.
- Expect:
  `.codex-goal-verifier.json` contains
  `current_state.dashboard_fallback_category=live_verified_runtime_read_models`
  and `dashboard_read_models.knowledge_pipelines_table=true` when the dashboard
  knowledge/RAG verifier succeeds on the first or second attempt.

### Scenario 2

- Action:
  Run `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-dashboard-knowledge-rag-state-boundary.ps1`
  immediately after the unified verifier.
- Expect:
  If the standalone verifier returns `live_verified`, the unified artifact must
  not remain downgraded to `partially_live_verified_read_model` because of a
  transient first sample.

## Failure Signals

- `.codex-goal-verifier.json` still reports
  `dashboard_fallback_category=partially_live_verified_read_model` while the
  standalone dashboard knowledge/RAG verifier is `live_verified` on the same
  turn.

## Evidence

- `.\.codex-goal-verifier.json`
- stdout from:
  - `scripts/verify-go-migration-goal.ps1`
  - `scripts/verify-dashboard-knowledge-rag-state-boundary.ps1`
