# 154. Go Outbox Scope Live Verification Runbook

Date: 2026-06-02

## Status

Accepted

## Context

当前 Go local outbox worker 已经不是纯只读 readiness：

- 全局 `text` 已真实由 Go owner 执行
- 第二账号 `2365524513` 的 `group file` 已真实由 Go owner 执行
- 第一账号 `1049511700` 的 `private file` 已真实由 Go owner 执行
- 第一账号 `group file` 仍必须保持 gated
- 所有 `image` 仍必须保持 gated

这些边界之前都已经被单次 smoke 证明过，但缺少一个 repo-owned、
可重复执行的 live 验证入口。仅看 `/v1/queue-backend` 和
`/v1/outbound-cutover/readiness` 不能证明：

1. 已放开的路由现在仍然真的能自动成功
2. 未放开的路由现在仍然真的不会被错误执行

## Decision

新增 repo-owned live smoke 脚本：

- `scripts/verify-go-outbox-scope-live.ps1`

该脚本必须：

1. 读取当前 `/v1/outbound-cutover/readiness` 与 `/v1/queue-backend`
2. 创建真实 outbox 事件并轮询 `/v1/outbox/{event_id}`
3. 覆盖至少四个当前关键路由：
   - 第一账号 `private file` -> 预期 `succeeded`
   - 第二账号 `group file` -> 预期 `succeeded`
   - 第一账号 `group file` -> 预期保持 `queued/attempts=0`
   - 第一账号 `private image` -> 预期保持 `queued/attempts=0`
4. 输出单个 JSON 文档，记录：
   - readiness / queue backend 摘要
   - 每个 case 的 `event_id`
   - 最终状态
   - 轮询历史

该脚本是 live smoke，不是只读诊断；它会真实发送当前已放开的 file 路由。

## Consequences

正面：

- 当前 Go default owner 的实际边界变成可复跑证据，而不是只保留在历史日志或文档里。
- 后续 goal 更新可以直接引用同一 smoke 入口。
- 可直接证明 gate 仍在生效，而不是只证明配置字符串没变。

负面：

- 该脚本会真实向 QQ 发送已放开的 file 消息，因此不能默认在所有只读验证中自动执行。
- 它不解决 rich-media 平台 blocker，只验证当前 gate 和已放开路由是否仍符合预期。

## Verification

运行：

```powershell
.\scripts\verify-go-outbox-scope-live.ps1
```

期望结果：

- `first_account_private_file` 最终 `succeeded`
- `second_account_group_file` 最终 `succeeded`
- `first_account_group_file_gated` 最终保持 `queued/attempts=0`
- `first_account_private_image_gated` 最终保持 `queued/attempts=0`

后续如需在统一 goal 验证里一起执行，可通过
`verify-go-migration-goal.ps1 -IncludeOutboxScopeSmoke` 显式启用。
