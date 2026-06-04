# 185 Dashboard Observe Capture Table

## Context

`/v1/runtime-overview` 已透传 Go-owned `observe_capture` detail，但 runtime overview dashboard panel 仍主要依赖 raw JSON，不利于直接检查 observe-only 群的 capture 覆盖、receiver 近期活跃度和 media content readiness 边界。

## Decision

Python dashboard panel 为 `observe_capture` card 增加结构化只读 drilldown，直接展示：

- `targets/ready/warning/blocked/image_covered/file_covered/content_ready_assets/receiver_activity_recent`
- `targets` 的 `target_id/channel/status/receiver_status/receiver_activity_recent/inbox_events/media_assets/content_forbidden_assets/coverage.file_seen/coverage.media_content_ready`
- `notes`

## Constraints

- 只读展示，不新增任何 mutation UI
- 不修改 observe capture / inbox / media 记录
- 不创建、租约或执行 outbox / agent-job
- 不发送 QQ/Telegram
- 不触发 Python AI

## Verification

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
