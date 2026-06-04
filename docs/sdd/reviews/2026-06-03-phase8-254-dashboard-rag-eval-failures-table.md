# Phase 8.254 Review - Dashboard RAG Eval Failures Table

## Completed

- runtime overview dashboard panel 已为 `rag_eval_failures` 增加结构化只读
  drilldown。
- renderer 已兼容 dashboard fixture `detail.items` 与当前 live runtime 的
  `detail.agent_job_metrics` 形状。
- unified goal verifier 已把 `rag_eval_failures` 的 dashboard panel 证据并入
  `dashboard_read_models`。

## Evidence

- `GET /v1/runtime-overview`
  - `rag_eval_failures` card: `value=0`, `status=ok`
  - current live detail shape: `rag_eval_failures.detail.agent_job_metrics`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"` => `5 passed`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q` => `4 passed`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
  - `dashboard_read_models.rag_eval_failures_table=true`
  - `residual_classification.dashboard_fallback.category=live_verified_runtime_read_models`
  - `checks.goal_ready_to_close=false`
  - `checks.open_blockers=[telegram_token_missing, qq_image_native_platform_blocker_unresolved]`

## Remaining

- Telegram backend 仍缺 token
- QQ `image` 仍是 native 平台 blocker
