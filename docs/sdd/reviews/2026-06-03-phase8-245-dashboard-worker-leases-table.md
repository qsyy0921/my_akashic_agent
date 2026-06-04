# Phase 8.245 Review - Dashboard Worker Leases Table

## Completed

- runtime overview dashboard panel 已为 `worker_leases` 增加结构化只读 drilldown。
- dashboard reader 现在会在旧 Go overview payload 缺失 `worker_leases` card 时补
  fallback card。
- unified goal verifier 已把该 read-model 纳入当前 turn 取证。

## Evidence

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"` => `5 passed`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q` => `4 passed`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
  - `dashboard_read_models.worker_leases_table=true`
  - `checks.goal_ready_to_close=false`
  - `checks.open_blockers=[telegram_token_missing, qq_image_native_platform_blocker_unresolved]`

## Remaining

- Telegram backend 仍缺 token
- QQ `image` 仍是 native 平台 blocker
