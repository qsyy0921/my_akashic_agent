# ATDD - Dashboard Job Events Outbox Events Table

## Scenario

当 operator 打开 runtime overview dashboard 并查看 `job_events` 与
`outbox_events` detail 时，应能直接看到 event totals 与 recent event rows，
而不是只能读取 raw JSON。

## Acceptance

1. panel 显示 `Job Events`。
2. panel 显示 `No job event totals sampled`、`No recent job events sampled`。
3. panel 显示 `Outbox Events`。
4. panel 显示 `No outbox event totals sampled`、`No recent outbox events sampled`。
5. panel 显示 `Event ID`、`Delivery / Event`、`Occurred At`。
6. unified goal verifier 返回：
   - `dashboard_read_models.job_events_table=true`
   - `dashboard_read_models.outbox_events_table=true`
