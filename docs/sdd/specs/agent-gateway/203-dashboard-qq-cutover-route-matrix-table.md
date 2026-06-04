# 203 Dashboard QQ Cutover Route Matrix Table

## Context

repo-owned `scripts/verify-qq-cutover-route-matrix.ps1` 已能在当前 turn 直接给出：

- `go_execution_owner_scope`
- `currently_sendable_routes`
- `policy_blocked_routes`
- `platform_blocker_routes`

但 runtime overview dashboard 之前还没有对应的结构化只读 drilldown，只能看
raw JSON 或单独跑 verifier，不能在主 panel 上直接读出当前 QQ cutover route
matrix。

## Decision

在 `plugins/runtime_overview/dashboard.py`、`dashboard_panel.ts` 和静态
`dashboard_panel.js` 中新增 `qq_cutover_route_matrix` 的结构化 read-model。

reader 侧要求：

- 从 `runtime_overview.queue_backend` 的
  `outbox_allowed_kinds* + outbox_execution_owner + outbox_execution_scope`
  派生 Go 当前接管的 QQ route scope；
- 从 `runtime_overview.runtime_config.delivery.qq_group_send_enabled` 投影当前
  `currently_sendable_routes` 与 `policy_blocked_routes`；
- 固定保留当前 rich-media blocker taxonomy：
  - 两账号 `private/group image`
  - 第一账号 `1049511700` 的 `group file`
- 当 Go overview payload 缺少 `qq_cutover_route_matrix` card 时，由 dashboard
  reader 自动补 fallback card。

panel 侧要求：

- KPI 直接展示：
  - `go scope`
  - `currently sendable`
  - `policy blocked`
  - `platform blockers`
  - `group send enabled`
  - `execution scope`
- 结构化 detail 展示：
  - `Execution Owner`
  - `QQ Accounts`
  - `Go Execution Owner Scope` 表
  - `Currently Sendable Routes` 表
  - `Policy Blocked Routes` 表
  - `Platform Blocker Routes` 表
  - `notes`

同时把这条 panel 资产证据接进 `scripts/verify-go-migration-goal.ps1` 的
`dashboard_read_models`。

## Non-Goals

- 不重新开启 QQ 群发。
- 不改变 Go outbox route gate。
- 不执行新的 QQ rich-media native probe。
- 不把 route-matrix verifier 的判定逻辑搬进 dashboard 控制路径。

## Verification

- `GET /v1/runtime-overview`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
