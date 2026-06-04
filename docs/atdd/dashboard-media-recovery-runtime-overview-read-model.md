# ATDD: dashboard media recovery runtime-overview read model

## Scope

- dashboard `/api/dashboard/runtime-overview` 对 Go-owned
  `media_asset_content_recovery` summary/card/detail 的只读透传。

## Preconditions

- Go runtime 在 `http://127.0.0.1:8780` 可访问。
- Python dashboard 在 `http://127.0.0.1:2236` 可访问。
- repo 内脚本可执行：
  - `scripts/verify-media-recovery-boundary.ps1`
  - `scripts/verify-go-migration-goal.ps1`

## Scenarios

### Scenario 1

- Action:
  请求 `GET /v1/runtime-overview` 和
  `GET /api/dashboard/runtime-overview`。
- Expect:
  dashboard payload 同时包含：
  - `media_asset_content_recovery` top-level detail；
  - `cards[].id == "media_asset_content_recovery"`；
  - detail `reason` 与 Go runtime 一致。

### Scenario 2

- Action:
  运行 `.\scripts\verify-media-recovery-boundary.ps1`。
- Expect:
  输出中：
  - `dashboard.runtime_overview_has_media_content_recovery_card=true`
  - `dashboard.runtime_overview_has_media_content_recovery_detail=true`
  - `checks.dashboard_runtime_overview_missing_media_recovery_detail=false`

### Scenario 3

- Action:
  运行 `.\scripts\verify-go-migration-goal.ps1`。
- Expect:
  unified verifier 继续把 media recovery 归类为
  `go_control_plane_live_operator_runtime_config_boundary`，而不是 dashboard
  read-model gap。

## Failure Signals

- dashboard runtime-overview 没有 `media_asset_content_recovery` detail；
- 没有 `media_asset_content_recovery` card；
- detail `reason` 退回 `unknown`，但 Go runtime 已返回真实 detail；
- verifier 仍报告 dashboard runtime-overview 缺失 media recovery detail。

## Evidence

- `GET http://127.0.0.1:2236/api/dashboard/runtime-overview`
- `GET http://127.0.0.1:8780/v1/runtime-overview`
- `.\scripts\verify-media-recovery-boundary.ps1`
- `.\scripts\verify-go-migration-goal.ps1`
