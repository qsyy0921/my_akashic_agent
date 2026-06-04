# 169 Dashboard Queue Backend Table

## Context

`/v1/runtime-overview` 已透传 Go-owned `queue_backend` detail，但 runtime overview dashboard panel 仍主要依赖 raw JSON，不利于直接识别当前 MQ provider、execution owner 与 capability matrix。

## Decision

Python dashboard panel 为 `queue_backend` card 增加结构化只读 drilldown，直接展示：

- `provider/mode/migration_phase`
- `outbox_execution_owner/outbox_execution_scope`
- `agent_job_execution_owner`
- `recommended_first_backend`
- selected provider 对 `external_lease/result_ack` 的支持
- provider capability matrix
- notes

## Constraints

- 只读展示，不新增任何 mutation UI
- 不创建 MQ 连接，不 publish/lease/ack/nack/term
- 不改变 outbox owner、不改变 QQ 群发策略、不触发 Python AI

## Verification

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
