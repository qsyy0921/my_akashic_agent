# 173 Dashboard External Lease Diagnostics Table

## Context

`/v1/runtime-overview` 已透传 Go-owned `external_lease_diagnostics` detail，但 runtime overview dashboard panel 仍主要依赖 raw JSON，不利于直接检查 external queue 是否启用、selected provider capability、result-ack 支持边界和当前 execution owner。

## Decision

Python dashboard panel 为 `external_lease_diagnostics` card 增加结构化只读 drilldown，直接展示：

- `provider/mode/external_queue_configured/external_queue_active/consumer_concurrency/max_in_flight/outbox_execution_owner/agent_job_execution_owner`
- selected provider 的 `consumer_model/adapter_boundary/recommended_first_backend/supports_external_lease/supports_agent_job_result_ack/blockers`
- notes

## Constraints

- 只读展示，不新增任何 mutation UI
- 不修改 MQ provider、mode、DSN 或 cutover flags
- 不 publish/lease/ack/nack/term MQ work
- 不创建/执行 AgentJob 或 outbox delivery
- 不触发 QQ/Telegram 或 Python AI

## Verification

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
