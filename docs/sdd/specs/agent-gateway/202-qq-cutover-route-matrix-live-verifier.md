# SDD: QQ Cutover Route Matrix Live Verifier

## Problem

当前统一 goal verifier 只能给出 QQ cutover 的 `allowed_routes` 和
`unresolved_routes` 粗粒度摘要，不能直接回答这三个当前运行态问题：

1. 哪些 route 已经在 Go execution owner scope 内；
2. 哪些 route 只是因为 `qq_group_send_enabled=false` 被策略拦截；
3. 哪些 rich-media route 仍因为平台 blocker 被有意留在 Go scope 之外。

这会让每轮 live 盘点仍然需要人工重组 queue backend、runtime config 和
rich-media blocker 结论。

## Non-Goals

- 不重新开启 QQ 群发。
- 不执行新的 QQ rich-media native probe。
- 不改 Go runtime route gate 本身。
- 不把这类 route matrix 写成 dashboard 控制逻辑。

## Requirements

### Inputs

- `GET /v1/runtime-config`
- `GET /v1/queue-backend`
- `POST /v1/outbound-cutover/readiness`
- `GET /v1/queue-topology`
- `GET /v1/runtime-overview`

### Output

新增 repo-owned verifier：

- `scripts/verify_qq_cutover_route_matrix.py`
- `scripts/verify-qq-cutover-route-matrix.ps1`

输出必须包含：

- `go_execution_owner_scope`
- `currently_sendable_routes`
- `policy_blocked_routes`
- `platform_blocker_routes`
- `checks`
- `conclusion`

### Route Matrix Rules

- `go_execution_owner_scope` 基于当前 `queue-backend` 的
  `outbox_allowed_kinds*` 族字段构造，保留 wildcard scope。
- `currently_sendable_routes` 在不修改 runtime 配置的前提下，按
  `qq_group_send_enabled` 投影当前真正可发送的 route classes。
- `policy_blocked_routes` 只表示“本来在 Go owner scope 内，但当前被
  `qq_group_send_enabled=false` 阻断”的 group route classes。
- `platform_blocker_routes` 表示当前仍刻意不放入 Go scope 的 rich-media
  route classes：
  - 两个账号的 `image` 在 `private/group` 两类会话中都保持 blocker；
  - 第一账号 `1049511700` 的 `group file` 保持 blocker。

### Checks

verifier 必须校验：

- `qq_group_send_enabled=false`
- `outbox_execution_owner=go_local_outbox_worker`
- `outbox_execution_scope` 仍是 gated scope
- `outbound-cutover/readiness.execution_ready=true`
- `queue-topology` 和 `runtime-overview.summary` 与 queue backend 的 outbox owner
  一致
- global `text` 在 Go scope 中
- 第二账号 `file` 在 Go scope 中
- 第一账号 `private file` 在 Go scope 中
- 第一账号 `group file` 不在 Go scope 中
- 所有 `image` route 都不在 Go scope 中

## Invariants

- verifier 只读，不创建 outbox、不过滤/变更 runtime state、不发送 QQ 消息。
- verifier 不得把“被群发总开关阻断”和“仍因 rich-media blocker 被 gated”混为一类。
- unified goal verifier 必须消费新的 route matrix，而不是继续只保留
  `allowed_routes + unresolved_routes`。

## Acceptance

- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-qq-cutover-route-matrix.ps1`
  返回 `conclusion.status=live_verified`。
- `scripts/verify-go-migration-goal.ps1` 输出中包含
  `qq_cutover_route_matrix`，且 `qq_outbox_cutover.route_matrix`
  直接透传三类 route。
