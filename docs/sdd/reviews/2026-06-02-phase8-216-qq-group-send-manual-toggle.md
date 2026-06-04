# Phase 8.216 Review - QQ Group Send Manual Toggle

## Summary

- 把 QQ 群发送从“observe-only 群 hard block”扩展成“全局 QQ group 可手动开关”的策略。
- 当前本地默认值已切到关闭：
  - Go: `AKASHIC_QQ_GROUP_SEND_ENABLED=false`
  - Python: `group_send_enabled=false`

## Verified

- Go runtime live:
  - `GET /v1/runtime-config` 返回 `delivery.qq_group_send_enabled=false`
  - `scripts/verify-go-outbox-scope-live.ps1` 真实证明：
    - 第一账号 group text 被阻断
    - 第二账号 group file 被阻断
    - 第一账号 private file 仍成功
- Unified goal verifier:
  - `scripts/verify-go-migration-goal.ps1` 真实输出：
    - `qq_group_send_toggle_disabled=true`
    - `qq_group_route_blocked=true`
    - `qq_group_route_blocked_by_global_policy=true`
    - `private_route_still_ready=true`
- Code tests:
  - `go test ./app/service ./cmd/agent-runtime`
  - `uv run pytest tests/test_channel_clients.py::test_qq_channel_paths tests/test_channel_clients.py::test_qq_channel_records_runtime_send_ledger tests/test_bootstrap_wiring_p2.py::test_config_load_reads_qq_group_send_toggle tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

## Residual Risk

- 旧 Python worker lease 在重启后的短窗口内仍可能产生一次性 `409 lease conflict` 日志，但不会重新打开 QQ group send。
- Telegram token 缺失与 QQ image native blocker 仍是整体 goal 的剩余 blocker，但与本切片无关。
