# Phase 8.250 Review - Dashboard Dead Letters Table

## Completed

- runtime overview dashboard panel 已为 `dead_letters` 增加结构化只读 drilldown。
- unified goal verifier 已把这条 read-model 纳入当前 turn 取证。

## Evidence

- `GET /v1/runtime-overview`
  - `dead_letters` card: `value=102`, `status=danger`
  - live detail:
    - `agent_job_metrics.dead_letters.current_total=0`
    - `outbox_metrics.dead_letters.current_total=102`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"` => `5 passed`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q` => `4 passed`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
  - `dashboard_read_models.dead_letters_table=true`
  - `checks.goal_ready_to_close=false`
  - `checks.open_blockers=[telegram_token_missing, qq_image_native_platform_blocker_unresolved]`

## Remaining

- Telegram backend 仍缺 token
- QQ `image` 仍是 native 平台 blocker
