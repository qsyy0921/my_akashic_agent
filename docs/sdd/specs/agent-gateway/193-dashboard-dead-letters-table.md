# 193 Dashboard Dead Letters Table

## Context

`/v1/runtime-overview` 已提供 `dead_letters` card/detail，但 dashboard panel
之前只能通过 raw JSON 查看聚合后的 AgentJob / outbox dead-letter 诊断。

## Decision

在 `plugins/runtime_overview/dashboard_panel.ts` 为 `dead_letters` 增加结构化只读
drilldown，展示：

- `agent job dead letters / outbox dead letters / agent job types / outbox channel kinds`
- AgentJob dead-letter recent 表
- outbox dead-letter recent 表
- AgentJob / outbox dead-letter totals

同时将这条 read-model 纳入 unified goal verifier 的当前 turn 取证。

## Non-Goals

- 不 ack / nack / term MQ
- 不重试 AgentJob 或 outbox delivery
- 不修改 queue owner / delivery policy
- 不触发 Python AI

## Verification

- `GET /v1/runtime-overview`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
