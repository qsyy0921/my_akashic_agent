# 191 Dashboard Media Asset Retention Plan Cleanup Table

## Context

`/v1/runtime-overview` 已提供 `media_asset_retention_plan` 与
`media_asset_retention_cleanup` card/detail，但 dashboard panel 之前只能通过 raw
JSON 查看 retention candidate、required steps、cleanup totals 和 notes。

## Decision

在 `plugins/runtime_overview/dashboard_panel.ts` 为
`media_asset_retention_plan` 与 `media_asset_retention_cleanup` 增加结构化只读
drilldown，展示：

- retention plan 的 `ready/assets/candidates/required steps/verify steps/rollback steps`
- retention plan 的 blockers、diagnostics assets 与 required/verification/rollback
  steps 表格
- retention cleanup 的 `ready/assets/candidates/planned/applied/failed/audits/rolled back`
- retention cleanup 的 endpoints、blockers 与 notes 表格

同时将这两条 read-model 纳入 unified goal verifier 的当前 turn 取证。

## Non-Goals

- 不执行 retention cleanup
- 不创建 approval / control mutation
- 不删除 media metadata 或本地文件
- 不触发 OCR / VLM / RAG / AI

## Verification

- `GET /v1/runtime-overview`
- `GET /v1/media-assets/retention-plan?limit=50`
- `GET /v1/media-assets/retention-diagnostics?limit=50`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
