# Review: QQ Cutover Route Matrix Live Verifier

## Scope

- 新增 repo-owned `qq_cutover_route_matrix` live verifier
- 将该 verifier 接入 unified goal verifier

## What Changed

- 新增 `scripts/verify_qq_cutover_route_matrix.py`
- 新增 `scripts/verify-qq-cutover-route-matrix.ps1`
- `scripts/verify-go-migration-goal.ps1` 现已直接消费 route matrix，
  不再只靠 `allowed_routes + unresolved_routes`

## Checks

- verifier 只读，不重新开启 QQ 群发，不执行新的 native rich-media probe
- route matrix 把当前 runtime 切成三类：
  - Go owner scope
  - group-send policy blocked
  - platform blocker gated
- unified goal verifier 直接透传该结构，便于后续每轮复核

## Residual Risk

- `platform_blocker_routes` 仍依赖当前 blocker taxonomy，而不是在本 verifier 中
  重新执行 native probe；这符合“不擅自恢复群发/不额外发消息”的当前边界。
- rich-media 是否真正恢复，仍以独立 native/live smoke 为准。
