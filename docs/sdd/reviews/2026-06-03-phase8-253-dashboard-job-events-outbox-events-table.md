# Phase 8.253 Review - Dashboard Job Events Outbox Events Table

## Completed

- runtime overview dashboard panel 已为 `job_events` 增加结构化只读 drilldown。
- runtime overview dashboard panel 已为 `outbox_events` 增加结构化只读 drilldown。
- renderer 已兼容旧 `recent_events` detail 与当前 live metrics-nested detail。
- unified goal verifier 已把这两条 read-model 纳入当前 turn 取证。

## Evidence

- `GET /v1/runtime-overview`
  - `job_events` card: `value=50`, `status=ok`
  - `outbox_events` card: `value=50`, `status=ok`
  - current live detail shape:
    - `job_events.detail.agent_job_metrics`
    - `outbox_events.detail.outbox_metrics`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"` => `5 passed`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
  - `dashboard_read_models.job_events_table=true`
  - `dashboard_read_models.outbox_events_table=true`
  - `checks.goal_ready_to_close=false`
  - `checks.open_blockers=[telegram_token_missing, qq_image_native_platform_blocker_unresolved]`

## Remaining

- Telegram backend 仍缺 token
- QQ `image` 仍是 native 平台 blocker
- dashboard remaining raw-JSON-only gap 缩到 `rag_eval_failures`
