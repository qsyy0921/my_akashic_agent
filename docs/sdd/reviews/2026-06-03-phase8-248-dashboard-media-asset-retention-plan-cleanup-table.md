# Phase 8.248 Review - Dashboard Media Asset Retention Plan Cleanup Table

## Completed

- runtime overview dashboard panel 已为 `media_asset_retention_plan` 与
  `media_asset_retention_cleanup` 增加结构化只读 drilldown。
- unified goal verifier 已把这两条 read-model 纳入当前 turn 取证。

## Evidence

- `GET /v1/runtime-overview`
  - `media_asset_retention_plan` card:
    `value=media_asset_retention_no_cleanup_candidates:1`, `status=ok`
  - `media_asset_retention_cleanup` card: `value=0/0`, `status=muted`
- `GET /v1/media-assets/retention-plan?limit=50`
  - `ready=false`
  - `reason=media_asset_retention_no_cleanup_candidates`
  - `candidate_count=0`
- `GET /v1/media-assets/retention-diagnostics?limit=50`
  - `totals.assets=50`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"` => `5 passed`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q` => `4 passed`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
  - `dashboard_read_models.media_asset_retention_plan_table=true`
  - `dashboard_read_models.media_asset_retention_cleanup_table=true`
  - `checks.goal_ready_to_close=false`
  - `checks.open_blockers=[telegram_token_missing, qq_image_native_platform_blocker_unresolved]`

## Remaining

- Telegram backend 仍缺 token
- QQ `image` 仍是 native 平台 blocker
