# Phase 8.247 Review - Dashboard Knowledge Planner Preview Readiness Table

## Completed

- runtime overview dashboard panel 已为 `knowledge_job_planner_preview` 与
  `knowledge_job_planner_readiness` 增加结构化只读 drilldown。
- unified goal verifier 已把这两条 read-model 纳入当前 turn 取证。

## Evidence

- `GET /v1/runtime-overview`
  - `knowledge_job_planner_preview` card: `value=6/6`, `status=ok`
  - `knowledge_job_planner_readiness` card: `value=ready`, `status=ok`
- `GET /v1/knowledge-job-planner/readiness`
  - `ready=true`
  - `planner_enabled=true`
  - `planner_running=true`
  - `knowledge_worker_ready=true`
- `GET /v1/knowledge-job-planner/cutover-plan`
  - `current_admission_owner=go_runtime_knowledge_job_planner`
  - `decision=ready`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"` => `5 passed`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q` => `4 passed`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
  - `dashboard_read_models.knowledge_job_planner_preview_table=true`
  - `dashboard_read_models.knowledge_job_planner_readiness_table=true`
  - `checks.goal_ready_to_close=false`
  - `checks.open_blockers=[telegram_token_missing, qq_image_native_platform_blocker_unresolved]`

## Remaining

- Telegram backend 仍缺 token
- QQ `image` 仍是 native 平台 blocker
