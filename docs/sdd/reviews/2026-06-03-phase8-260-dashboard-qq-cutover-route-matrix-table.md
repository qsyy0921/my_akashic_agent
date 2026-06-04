# Review: Dashboard QQ Cutover Route Matrix Table

## Scope

- 为 runtime overview dashboard 增加 `qq_cutover_route_matrix` 结构化只读
  drilldown
- 将该 panel 资产接入 unified goal verifier

## What Changed

- `dashboard.py` 现在会从 `queue_backend + runtime_config` 派生
  `qq_cutover_route_matrix`
- `dashboard_panel.ts/.js` 现在直接展示 route-matrix KPI 和四类 route 表
- `verify-go-migration-goal.ps1` 现在把
  `dashboard_read_models.qq_cutover_route_matrix_table` 纳入取证
- 聚合测试补了 route totals、top-level payload 和 panel 资产断言

## Checks

- 只读，不开启 QQ 群发，不改 outbox gate，不执行 native probe
- 当前 route matrix 仍保持三类边界：
  - Go 已接管
  - 群发策略阻断
  - rich-media 平台 blocker
- queue backend normalization 不再丢失 `outbox_execution_owner/scope` 和
  `outbox_allowed_kinds*`

## Residual Risk

- `platform_blocker_routes` 仍依赖当前 blocker taxonomy，而不是在 dashboard
  路径重新执行 live probe；这是有意保持只读边界。
- 若未来 QQ route gate 字段形状再变化，需要同步 reader derivation 和聚合测试。
