# Go Outbox Scope Live Verification ATDD

Date: 2026-06-02

## Goal

用真实 outbox 事件验证当前 Go default owner 已放开的路由仍成功、未放开的路由仍被 gate。

## Acceptance Scenarios

### 1. 已放开的 file 路由真实成功

前置条件：

- `agent-runtime` 正在运行
- 当前 runtime 为 `account_conversation_kind_gated`

步骤：

1. 运行 `.\scripts\verify-go-outbox-scope-live.ps1`

期望结果：

- `first_account_private_file` 最终 `succeeded`
- `second_account_group_file` 最终 `succeeded`

### 2. 未放开的 route 仍保持 gated

前置条件：

- 同上

步骤：

1. 运行 `.\scripts\verify-go-outbox-scope-live.ps1`

期望结果：

- `first_account_group_file_gated` 保持 `queued/attempts=0`
- `first_account_private_image_gated` 保持 `queued/attempts=0`

### 3. 统一 goal 验证显式启用 live smoke

前置条件：

- 允许本轮发出真实 file smoke

步骤：

1. 运行 `.\scripts\verify-go-migration-goal.ps1 -IncludeOutboxScopeSmoke`

期望结果：

- `qq_outbox_cutover.live_scope_smoke` 存在
- `qq_outbox_cutover.conclusion=partial_go_default_owner_live_verified`
