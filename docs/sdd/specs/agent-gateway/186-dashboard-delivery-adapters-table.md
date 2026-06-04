# 186 Dashboard Delivery Adapters Table

## Context

`/v1/runtime-overview` 已透传 Go-owned `delivery_adapters` detail，但 runtime overview dashboard panel 仍主要依赖 raw JSON，不利于直接检查当前 QQ OneBot adapter 是否齐全、endpoint/token 配置是否到位，以及 Telegram adapter 当前为何未出现。

## Decision

Python dashboard panel 为 `delivery_adapters` card 增加结构化只读 drilldown，直接展示：

- `provider/channel/transport/enabled/endpoint_configured/access_token_configured/endpoint`
- 保留现有 `Adapter Health` 手动 probe 和 `Delivery Smoke` 只读入口

## Constraints

- 只读展示，不新增任何 mutation UI
- 不修改 adapter 配置
- 不发送 QQ/Telegram
- 不触发 Python AI

## Verification

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
