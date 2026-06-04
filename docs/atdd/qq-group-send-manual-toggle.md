# QQ Group Send Manual Toggle ATDD

## Scenario

本地 runtime 需要支持手动关闭或开启 QQ 群发送；当前默认关闭。

## Preconditions

- `scripts/start-agent-runtime.ps1` 以 `-QQGroupSendEnabled false` 启动
- `config.toml` 中两个 QQ 配置的 `group_send_enabled=false`
- QQ OneBot receiver 正常连接

## Acceptance

1. `GET /v1/runtime-config` 返回 `delivery.qq_group_send_enabled=false`
2. 对任意 `qq/group` route 执行 `delivery-dispatch/readiness`：
   - 返回 `HTTP 400`
   - `message="qq group sends are disabled"`
   - `data.error_kind="route_error"`
3. 对 `qq/private` route 执行 `delivery-dispatch/readiness`：
   - 返回 `ready=true`
4. `scripts/verify-go-outbox-scope-live.ps1` 中 group case 被阻断，private file case 成功
5. `scripts/verify-go-migration-goal.ps1` 默认输出中包含：
   - `qq_group_send_toggle_disabled=true`
   - `qq_group_route_blocked=true`
   - `private_route_still_ready=true`

## Failure Signals

- 任意 QQ 群 route 仍可进入 `ready=true`
- Python channel 仍向群发 text/file/image
- private route 也被误伤
