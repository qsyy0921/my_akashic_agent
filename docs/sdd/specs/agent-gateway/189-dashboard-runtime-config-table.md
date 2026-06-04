# 189 Dashboard Runtime Config Table

## Context

`/v1/runtime-overview` 已提供 `runtime_config` card 和 sanitized config detail，但
dashboard panel 之前只能靠 raw JSON 看 runtime 地址、QQ group send toggle、
Telegram token 配置、OneBot endpoint 和 worker flags。

## Decision

在 `plugins/runtime_overview/dashboard_panel.ts` 为 `runtime_config` 增加结构化只读
drilldown，展示：

- `QQ Group Send`
- `Telegram Token`
- `OneBot Endpoints`
- `Expected Channels`
- `Strict Lease Token`
- `AgentJob External Lease`
- Runtime / delivery / worker flags / environment keys 的表格

同时在 dashboard reader 的 Go overview 归一化路径上增加 fallback：如果旧 payload
缺少 `runtime_config` card，则基于 top-level `runtime_config` 追加一张只读 card，
并把 `runtime_base_url` 规范化成 `host:port` 展示值，保持与 live card 一致。

## Non-Goals

- 不修改 runtime 配置
- 不补 token / endpoint / worker flag
- 不触发 QQ/Telegram 发送或 Python AI

## Verification

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
