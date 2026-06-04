# Phase 8.227 Review - Dashboard Receiver Statuses Table

## Completed

- runtime overview dashboard panel 已为 `receiver_statuses` card 增加结构化只读 drilldown。
- unified goal verifier 已把该 read-model 纳入当前 turn 取证。

## Evidence

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"` => `5 passed`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
  - `dashboard_read_models.receiver_statuses_table=true`
  - `residual_classification.dashboard_fallback.category=live_verified_runtime_read_models`

## Remaining

- Telegram backend 仍缺 token
- QQ `image` 仍是 native 平台 blocker
