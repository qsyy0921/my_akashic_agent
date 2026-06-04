# 181 Dashboard Send Ledger Metrics Table

## Context

`/v1/runtime-overview` 已透传 Go-owned `send_ledger_metrics` detail，但 runtime overview dashboard panel 仍主要依赖 raw JSON，不利于直接检查重复 content hash、最近 ledger 记录和 bot-to-bot 循环风险。

## Decision

Python dashboard panel 为 `send_ledger_metrics` card 增加结构化只读 drilldown，直接展示：

- `sampled_records/unique_bots/unique_conversations/unique_content_hashes/repeated_content_hashes`
- `repeated_hashes` 的 `from_bot_id/conversation_id/content_hash/count/latest_timestamp`
- `recent` 的 `from_bot_id/conversation_id/content_hash/timestamp`

## Constraints

- 只读展示，不新增任何 mutation UI
- 不创建、租约或执行 outbox delivery
- 不修改 loop guard / send ledger 状态
- 不发送 QQ/Telegram
- 不触发 Python AI

## Verification

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
