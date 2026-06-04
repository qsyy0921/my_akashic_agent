# Review: Dashboard Knowledge RAG State Boundary Live Verifier

## What changed

- Added `scripts/verify_dashboard_knowledge_rag_state_boundary.py` and
  `scripts/verify-dashboard-knowledge-rag-state-boundary.ps1`.
- Strengthened `plugins/runtime_overview/dashboard.py` fallback so
  `knowledge_pipelines` top-level payload, summary, and card are synthesized
  from `/v1/knowledge-pipeline-diagnostics`.
- Increased the fallback diagnostics timeout to avoid empty live dashboard
  knowledge/RAG detail caused by a too-short default timeout.
- Wired the verifier into `scripts/verify-go-migration-goal.ps1` and its
  focused tests.

## Why

The existing `knowledge_pipelines` dashboard evidence was still too static.
It could prove the panel asset existed, but not that the current dashboard
runtime-overview still matched the live Go knowledge/RAG boundary.

## Evidence

- `uv run pytest tests/test_verify_dashboard_knowledge_rag_state_boundary.py tests/test_runtime_overview_dashboard_plugin.py tests/test_verify_go_migration_goal_script.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-dashboard-knowledge-rag-state-boundary.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
- The unified artifact now contains
  `dashboard_read_models.knowledge_pipelines_table=true` and
  `residual_classification.knowledge_rag_state_boundary.category=checkpoint_snapshot_derived_control_plane_live_verified`.
