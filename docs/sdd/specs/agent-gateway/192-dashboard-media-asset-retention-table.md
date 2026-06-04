# 192 Dashboard Media Asset Retention Table

## Context

`/v1/runtime-overview` 已提供 `media_asset_retention` card/detail，但 dashboard panel
之前只能通过 raw JSON 查看 retention totals、recent assets 和 notes。

## Decision

在 `plugins/runtime_overview/dashboard_panel.ts` 为 `media_asset_retention` 增加结构化
只读 drilldown，展示：

- `assets/cleanup_due/default/ephemeral/permanent/unknown`
- recent assets 的 `asset/name/retention class/cleanup due/age/cleanup after/reason/target`
- diagnostics notes

同时将这条 read-model 纳入 unified goal verifier 的当前 turn 取证。

## Non-Goals

- 不执行 retention cleanup
- 不创建 approval / control mutation
- 不删除 media metadata 或本地文件
- 不触发 OCR / VLM / RAG / AI

## Verification

- `GET /v1/runtime-overview`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
