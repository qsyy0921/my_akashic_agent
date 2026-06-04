# Phase 8.252 Review - Dashboard Runtime Health Stale Jobs Table

## Completed

- runtime overview dashboard panel 已为 `runtime_health` 增加结构化只读 drilldown。
- runtime overview dashboard panel 已为 `stale_jobs` 增加结构化只读 drilldown。
- unified goal verifier 已把这两条 read-model 纳入当前 turn 取证。

## Evidence

- `GET /v1/runtime-overview`
  - `runtime_health` card: `value=ok`, `status=ok`
  - `stale_jobs` card: `value=0`, `status=ok`
  - current stale-job diagnostics: `jobs=200`, `stale_leases=0`, `workers=3`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"` => `5 passed`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
  - `dashboard_read_models.runtime_health_table=true`
  - `dashboard_read_models.stale_jobs_table=true`
  - `checks.goal_ready_to_close=false`
  - `checks.open_blockers=[telegram_token_missing, qq_image_native_platform_blocker_unresolved]`

## Remaining

- Telegram backend 仍缺 token
- QQ `image` 仍是 native 平台 blocker
