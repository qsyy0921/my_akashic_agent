# ATDD: Dashboard Knowledge RAG State Boundary Live Verifier

## Scenario

As an operator, I need a repo-owned live verifier that proves the dashboard
runtime-overview still exposes the current Go knowledge-pipeline and
checkpoint-derived RAG dataset/index state on this turn.

## Acceptance

1. Run
   `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-dashboard-knowledge-rag-state-boundary.ps1`.
2. Confirm the result contains:
   - `runtime_overview_exposes_knowledge_pipelines_card=true`
   - `dashboard_overview_exposes_knowledge_pipelines_card=true`
   - `dashboard_card_matches_runtime_knowledge_pipeline_state=true`
   - `dashboard_top_level_knowledge_pipelines_matches_runtime=true`
   - `knowledge_pipeline_boundary_is_read_only=true`
3. Confirm the final conclusion is `live_verified`.
4. Rerun
   `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
   and confirm the current-turn artifact exposes:
   - `dashboard_read_models.knowledge_pipelines_table=true`
   - `checks.dashboard_read_models.knowledge_pipelines_table=true`
   - `residual_classification.knowledge_rag_state_boundary.category=checkpoint_snapshot_derived_control_plane_live_verified`
