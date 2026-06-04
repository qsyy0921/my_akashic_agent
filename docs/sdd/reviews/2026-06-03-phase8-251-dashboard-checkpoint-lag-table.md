# Phase 8.251 Review - Dashboard Checkpoint Lag Table

## Completed

- runtime overview dashboard panel 已为 `checkpoint_lag` 增加结构化只读 drilldown。
- unified goal verifier 已把这条 read-model 纳入当前 turn 取证。

## Evidence

- `GET /v1/runtime-overview`
  - `checkpoint_lag` card: `value=0`, `status=ok`
  - current detail keys: `diagnostics`
  - current live diagnostics show no lagged checkpoints
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"` => `5 passed`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q` => `4 passed`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
  - `dashboard_read_models.checkpoint_lag_table=true`
  - `checks.goal_ready_to_close=false`
  - `checks.open_blockers=[telegram_token_missing, qq_image_native_platform_blocker_unresolved]`

## Remaining

- Telegram backend 仍缺 token
- QQ `image` 仍是 native 平台 blocker
