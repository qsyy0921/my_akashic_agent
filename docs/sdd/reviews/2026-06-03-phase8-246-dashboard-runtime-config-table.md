# Phase 8.246 Review - Dashboard Runtime Config Table

## Completed

- runtime overview dashboard panel 已为 `runtime_config` 增加结构化只读 drilldown。
- dashboard reader 现在会在旧 Go overview payload 缺失 `runtime_config` card 时补
  fallback card，并把 fallback 地址统一规范成 `host:port`。
- unified goal verifier 已把该 read-model 纳入当前 turn 取证。

## Evidence

- `GET /v1/runtime-config`
  - `runtime.address=127.0.0.1:8780`
  - `delivery.qq_group_send_enabled=false`
  - `delivery.telegram_token_configured=false`
  - `workers.agent_job_strict_lease_token=false`
  - `workers.queue_external_lease_agent_job_enabled=false`
- `GET /v1/runtime-overview`
  - `runtime_config` card: `value=127.0.0.1:8780`, `status=ok`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"` => `5 passed`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q` => `4 passed`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
  - `dashboard_read_models.runtime_config_table=true`
  - `checks.goal_ready_to_close=false`
  - `checks.open_blockers=[telegram_token_missing, qq_image_native_platform_blocker_unresolved]`

## Remaining

- Telegram backend 仍缺 token
- QQ `image` 仍是 native 平台 blocker
