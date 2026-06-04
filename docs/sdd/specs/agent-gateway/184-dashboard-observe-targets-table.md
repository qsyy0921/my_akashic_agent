# 184 Dashboard Observe Targets Table

## Context

`/v1/runtime-overview` 已透传 Go-owned `observe_targets` detail，但 runtime overview dashboard panel 仍主要依赖 raw JSON，不利于直接检查 observe-only 配置、reply policy 和当前 target 分布。

## Decision

Python dashboard panel 为 `observe_targets` card 增加结构化只读 drilldown，直接展示：

- `targets/enabled/observe_only/reply_allowed/groups/qq`
- `targets` 的 `target_id/channel/conversation_type/observe_only/reply_allowed/require_at/enabled/source/updated_at`
- `notes`

## Constraints

- 只读展示，不新增任何 mutation UI
- 不修改 observe target 配置
- 不创建、租约或执行 outbox / agent-job
- 不发送 QQ/Telegram
- 不触发 Python AI

## Verification

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
