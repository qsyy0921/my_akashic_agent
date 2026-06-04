# 196 Dashboard Job Events Outbox Events Table

## Context

`/v1/runtime-overview` 已提供 `job_events` 与 `outbox_events` card/detail，但
dashboard panel 之前仍只能通过 raw JSON 查看 AgentJob / outbox event 读模型。

当前 live runtime 的 detail 形状与早期 dashboard fixture 不完全一致：

- 旧形状：`recent_events`
- 当前 live 形状：
  - `job_events.detail.agent_job_metrics`
  - `outbox_events.detail.outbox_metrics`

## Decision

在 `plugins/runtime_overview/dashboard_panel.ts` / `.js` 为 `job_events` 与
`outbox_events` 增加结构化只读 drilldown，并兼容上述两种 detail 形状。

展示内容：

- `job_events`
  - `sampled events/created/leased/running/succeeded/failed/terminal events`
  - event-type totals 表
  - recent job event 表
- `outbox_events`
  - `sampled events/queued/leased/dispatching/succeeded/failed/dead lettered/terminal events`
  - event-type totals 表
  - recent outbox event / dead-letter fallback 表

同时将这两条 read-model 纳入 unified goal verifier 的当前 turn 取证。

## Non-Goals

- 不创建 / 租约 / 重试 AgentJob
- 不 dispatch / cleanup / retry outbox delivery
- 不修改 worker、queue owner、lease、dead-letter 或 event store 状态
- 不触发 Python AI

## Verification

- `GET /v1/runtime-overview`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
