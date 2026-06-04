# 159 QQ Group Send Manual Toggle

## Context

- 用户要求两个 QQ 账号不要向任何 QQ 群发送消息。
- 之前 Go runtime 只对 observe-only 群做 hard block，不足以覆盖所有 QQ 群。
- 用户随后要求不要把这个行为写死，而是保留手动开启/关闭能力。

## Decision

- 为 QQ 群发送增加一个可显式切换的全局策略开关：
  - Go runtime 环境变量：`AKASHIC_QQ_GROUP_SEND_ENABLED=true|false`
  - Python QQ channel 配置：`group_send_enabled = true|false`
- 当前本地默认值设为 `false`，即默认静默所有 QQ 群发送。
- 当开关关闭时：
  - Go `delivery-dispatch/readiness` 对任意 `qq/group` route 返回
    `route_error`，消息为 `qq group sends are disabled`
  - Python QQ channel 对 group text / file / image 发送直接拒绝
  - group `/stop` 仍允许执行本地 interrupt，但不向群内回复

## Scope

- 覆盖两个 QQ 账号：
  - `1049511700`
  - `2365524513`
- 覆盖任意 QQ group route，而不是只覆盖 observe-only 群。
- QQ private route 不受影响。

## Non-Goals

- 不改变 Telegram 行为。
- 不改变 QQ private send gate。
- 不解决 QQ rich-media 平台 blocker。

## Live Verification

- `GET /v1/runtime-config` 显示 `delivery.qq_group_send_enabled=false`
- `scripts/verify-go-outbox-scope-live.ps1`：
  - `first_account_group_text_blocked` 被阻断
  - `second_account_group_file_blocked` 被阻断
  - private file 仍可成功
- `scripts/verify-go-migration-goal.ps1`：
  - `observe_only_silence.checks.qq_group_send_toggle_disabled=true`
  - `observe_only_silence.checks.qq_group_route_blocked=true`
  - `observe_only_silence.checks.qq_group_route_blocked_by_global_policy=true`
  - `observe_only_silence.checks.private_route_still_ready=true`

## Operator Entry Points

- Python 配置：
  - `config.toml`
  - `[channels.qq].group_send_enabled`
  - `[[channels.qq.accounts]].group_send_enabled`
- Go runtime 启动：
  - `scripts/start-agent-runtime.ps1 -QQGroupSendEnabled false|true`
  - 或环境变量 `AKASHIC_QQ_GROUP_SEND_ENABLED`
