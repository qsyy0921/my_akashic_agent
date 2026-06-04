# 171 Dashboard Receiver Leases Table

## Context

`/v1/runtime-overview` 已透传 Go-owned `receiver_leases` detail，但 runtime overview dashboard panel 仍主要依赖 raw JSON，不利于直接检查 Telegram polling conflict、expired lease cleanup 和接收端单实例控制。

## Decision

Python dashboard panel 为 `receiver_leases` card 增加结构化只读 drilldown，直接展示：

- `leases/active/expired/qq/telegram` totals
- 每个 lease 的 `receiver_id/kind/account_id/owner_instance_id/status/expires_at/updated_at`
- notes

## Constraints

- 只读展示，不新增任何 mutation UI
- 不执行 `cleanup-expired`
- 不创建、替换或删除 receiver lease
- 不启动、停止或重连 QQ/Telegram receiver

## Verification

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
