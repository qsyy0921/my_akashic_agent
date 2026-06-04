# ATDD: QQ Cutover Route Matrix Live Verifier

## Scope

- 验证当前 live runtime 是否能直接给出 QQ cutover 的 route matrix：
  已由 Go 接管、被 QQ 群发总开关拦截、以及因 rich-media blocker 仍保持 gated。

## Preconditions

- 本地 `agent-runtime` 运行在 `http://127.0.0.1:8780`
- 当前 runtime 保持 `qq_group_send_enabled=false`
- 当前 queue backend 仍为 `provider=local`
- 不重新开启 QQ 群发，不执行新的 native rich-media probe

## Scenarios

### Scenario 1

- Action:
  运行 `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-qq-cutover-route-matrix.ps1`
- Expect:
  返回 JSON，且 `route_matrix.go_execution_owner_scope` 包含：
  - global `text`
  - second-account `file`
  - first-account `private file`

### Scenario 2

- Action:
  查看 `route_matrix.policy_blocked_routes`
- Expect:
  当前 `qq_group_send_enabled=false` 时，group `text` 与 second-account
  `group file` 被明确列为 policy-blocked，而不是误报成 platform blocker。

### Scenario 3

- Action:
  查看 `route_matrix.platform_blocker_routes`
- Expect:
  当前包含：
  - 两个账号 `private/group image`
  - 第一账号 `group file`
  且这些 route 不会出现在 `go_execution_owner_scope`。

### Scenario 4

- Action:
  运行 `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
- Expect:
  输出中同时存在：
  - `qq_cutover_route_matrix`
  - `qq_outbox_cutover.route_matrix`
  - `residual_classification.qq_cutover_route_matrix`

## Failure Signals

- verifier 没有区分 policy-blocked 与 platform-blocker route。
- 第一账号 `group file` 或任意 `image` route 混入 `go_execution_owner_scope`。
- unified goal verifier 仍只输出 `allowed_routes/unresolved_routes` 粗粒度摘要。

## Evidence

- `scripts/verify-qq-cutover-route-matrix.ps1` 输出 JSON
- `scripts/verify-go-migration-goal.ps1` 输出 JSON
