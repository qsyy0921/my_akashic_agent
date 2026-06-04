# SDD: Goal Verifier Dashboard Knowledge/RAG Retry Hardening

## Problem

`scripts/verify-go-migration-goal.ps1` can occasionally write
`dashboard_fallback_category=partially_live_verified_read_model` even when the
repo-owned `verify-dashboard-knowledge-rag-state-boundary.ps1` verifier passes
immediately afterward on the same live runtime.

The user-visible problem is unstable current-turn evidence in
`.codex-goal-verifier.json`, not a missing Go control-plane capability.

## Non-goals

- Do not change Go runtime knowledge/RAG behavior.
- Do not change dashboard `knowledge_pipelines` read-model semantics.
- Do not introduce write-side retries or mutate runtime/dashboard state.

## Required Behavior

1. `verify-go-migration-goal.ps1` must still invoke the repo-owned dashboard
   knowledge/RAG verifier.
2. If the first verifier result is not `conclusion.status=live_verified`, the
   unified verifier must retry that verifier once after a short bounded delay.
3. If the retry returns `live_verified`, the unified verifier must use the
   retried result for:
   - `dashboard_knowledge_rag_state_boundary`
   - `residual_classification.knowledge_rag_state_boundary`
   - `dashboard_read_models.knowledge_pipelines_table`
   - `current_state.dashboard_fallback_category`
4. If the retry still fails, the unified verifier must preserve the failure and
   keep the artifact machine-readable.

## Invariants

- The hardening is read-only and may not modify runtime config, dashboard data,
  checkpoints, jobs, or RAG state.
- The retry count is exactly one extra attempt.
- The bounded delay must be short and local to this verifier path only.

## Acceptance

- A focused script test proves the retry branch exists in
  `verify-go-migration-goal.ps1`.
- Re-running `verify-go-migration-goal.ps1` after the hardening restores
  `current_state.dashboard_fallback_category=live_verified_runtime_read_models`
  when the live dashboard knowledge/RAG verifier passes on retry.
