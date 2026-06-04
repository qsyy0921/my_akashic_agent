# 183 Dashboard Inbound Dedupe Metrics Table

## Context

`/v1/runtime-overview` 已透传 Go-owned `inbound_dedupe_metrics` detail，但 runtime overview dashboard panel 仍主要依赖 raw JSON，不利于直接检查 dedupe scope、重复计数和最近 seen 覆盖。

## Decision

Python dashboard panel 为 `inbound_dedupe_metrics` card 增加结构化只读 drilldown，直接展示：

- `sampled_records/active_records/expired_records/duplicate_records/seen_total/duplicate_seen_total`
- `scopes` 的 `scope/records/active_records/expired_records/duplicate_records/seen_total/duplicate_seen_total/latest_seen_at`
- `notes`

## Constraints

- 只读展示，不新增任何 mutation UI
- 不修改 inbox dedupe 记录
- 不创建、租约或执行 outbox / agent-job
- 不发送 QQ/Telegram
- 不触发 Python AI

## Verification

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
