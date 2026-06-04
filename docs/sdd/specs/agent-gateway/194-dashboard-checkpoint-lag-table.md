# 194 Dashboard Checkpoint Lag Table

## Context

`/v1/runtime-overview` 已提供 `checkpoint_lag` card/detail，但 dashboard panel
之前只能通过 raw JSON 查看 lagged checkpoints 和 freshness 诊断。

## Decision

在 `plugins/runtime_overview/dashboard_panel.ts` 为 `checkpoint_lag` 增加结构化只读
drilldown，展示：

- `checkpoints/max lag/workers/stale after`
- lagged checkpoint 的 `session/job type/dataset/latest seq/checkpoint seq/lag messages/freshness/reason`

同时将这条 read-model 纳入 unified goal verifier 的当前 turn 取证。

## Non-Goals

- 不推进 checkpoint
- 不创建 / 重试 AgentJob
- 不修改 worker lease 或 queue owner
- 不触发 Python AI

## Verification

- `GET /v1/runtime-overview`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
