# QQ Group Send Manual Toggle TDD

## Targeted Tests

- Go:
  - `services/agent-runtime/app/service/delivery_dispatch_service_test.go`
  - 覆盖 `qqGroupSendEnabled=false` 时任意 `qq/group` route 返回 `route_error`
- Go:
  - `services/agent-runtime/cmd/agent-runtime/main_test.go`
  - 覆盖 `AKASHIC_QQ_GROUP_SEND_ENABLED` 出现在 runtime config 脱敏输出中
- Python:
  - `tests/test_channel_clients.py`
  - 覆盖 group text/file/image send 在 `group_send_enabled=false` 下被拒绝
  - 覆盖 group `/stop` 仍执行 interrupt 但不回群
- Python:
  - `tests/test_bootstrap_wiring_p2.py`
  - 覆盖 `config.toml` / account config 能读到 `group_send_enabled=false`

## Regression Focus

- 不要回退成只阻断 observe-only 群
- 不要误伤 private route
- 不要让 Python compatibility path 绕过 Go runtime gate
