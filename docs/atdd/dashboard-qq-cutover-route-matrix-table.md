# ATDD: Dashboard QQ Cutover Route Matrix Table

## Scope

- 验证 runtime overview dashboard 能直接结构化展示当前 QQ cutover route
  matrix，而不是只剩 raw JSON。

## Preconditions

- 本地 `agent-runtime` 运行在 `http://127.0.0.1:8780`
- 当前 runtime 保持 `qq_group_send_enabled=false`
- 当前 outbox owner 仍是 `go_local_outbox_worker`
- 当前 QQ rich-media blocker taxonomy 不变

## Scenarios

### Scenario 1

- Action:
  调用 `GET /v1/runtime-overview`
- Expect:
  返回的 overview summary 含有：
  - `qq_cutover_route_matrix_go_execution_owner_scope`
  - `qq_cutover_route_matrix_currently_sendable_routes`
  - `qq_cutover_route_matrix_policy_blocked_routes`
  - `qq_cutover_route_matrix_platform_blocker_routes`
  - `qq_cutover_route_matrix_group_send_enabled`

### Scenario 2

- Action:
  打开 runtime overview dashboard 的 `QQ Cutover Route Matrix` card/detail
- Expect:
  panel 直接展示四类 route 表：
  - `Go Execution Owner Scope`
  - `Currently Sendable Routes`
  - `Policy Blocked Routes`
  - `Platform Blocker Routes`

### Scenario 3

- Action:
  查看 `Policy Blocked Routes`
- Expect:
  当前 `qq/group` text 和 second-account `group file` 出现在 policy-blocked，
  而不是 platform blocker。

### Scenario 4

- Action:
  查看 `Platform Blocker Routes`
- Expect:
  当前包含两账号 `private/group image` 与 first-account `group file`，
  且这些 route 不混入 `Go Execution Owner Scope`。

### Scenario 5

- Action:
  运行 `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
- Expect:
  输出中 `dashboard_read_models.qq_cutover_route_matrix_table=true`。

## Failure Signals

- dashboard 仍只能看 raw JSON。
- `policy_blocked_routes` 和 `platform_blocker_routes` 被混淆。
- reader 丢失 `outbox_allowed_kinds*`，导致 route totals 变成 0。

## Evidence

- `/v1/runtime-overview` 返回体
- `tests/test_runtime_overview_dashboard_plugin.py`
- `scripts/verify-go-migration-goal.ps1` 输出 JSON
