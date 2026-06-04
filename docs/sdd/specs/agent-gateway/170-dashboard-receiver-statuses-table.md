# 170 Dashboard Receiver Statuses Table

## Context

`/v1/runtime-overview` 已透传 Go-owned `receiver_statuses` detail，但 runtime overview dashboard panel 仍主要依赖 raw JSON，不利于直接看清当前 QQ receiver 连接状态，以及 Telegram receiver 当前是否缺失。

## Decision

Python dashboard panel 为 `receiver_statuses` card 增加结构化只读 drilldown，直接展示：

- `receivers/connected/qq/telegram/suspended/failed` totals
- 每个 receiver 的 `receiver_id/kind/channel/account/status/reason/endpoint/updated_at`
- notes

## Constraints

- 只读展示，不新增任何 mutation UI
- 不创建或清理 receiver lease
- 不启动、停止或重连 QQ/Telegram receiver
- 不发送 QQ/Telegram 消息，不触发 Python AI

## Verification

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
