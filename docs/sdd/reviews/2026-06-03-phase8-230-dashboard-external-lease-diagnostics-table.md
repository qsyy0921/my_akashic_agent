# Phase 8.230 Review - Dashboard External Lease Diagnostics Table

## Completed

- runtime overview dashboard panel 已为 `external_lease_diagnostics` card 增加结构化只读 drilldown。
- unified goal verifier 已把该 read-model 纳入当前 turn 取证。

## Evidence

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"` => `5 passed`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
  - `dashboard_read_models.external_lease_diagnostics_table=true`
  - `residual_classification.dashboard_fallback.category=live_verified_runtime_read_models`

## Remaining

- Telegram backend 仍缺 token
- QQ `image` 仍是 native 平台 blocker
