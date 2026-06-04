# 168 Dashboard Delivery Smoke Readiness Table

## Context

`/v1/runtime-overview` 已经提供 Go-owned `delivery_smoke_readiness` detail，但 runtime overview dashboard panel 仍主要依赖 raw JSON，不利于直接识别当前 QQ/Telegram route blocker。

## Decision

Python dashboard panel 为 `delivery_smoke` card 增加结构化只读 drilldown，直接展示：

- overall `ready/reason/side_effect`
- `cases/ready/not_ready` totals
- 每个 smoke case 的 `name/ready/reason/channel/chat_id/conversation_type/kind/step_count`
- notes

## Constraints

- 只读展示，不新增任何 mutation UI
- 不主动发送 QQ/Telegram 消息
- 不调用 Go `delivery-smoke/readiness` 之外的新执行接口
- 不改变 QQ 群发关闭策略，也不绕过 observe-only 静默

## Verification

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
