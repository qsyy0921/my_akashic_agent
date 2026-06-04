# ATDD - Dashboard Runtime Health Stale Jobs Table

## Scenario

当 operator 打开 runtime overview dashboard 并查看 `runtime_health` 与
`stale_jobs` detail 时，应能直接看到 runtime health snapshot、health errors、
worker stale-job diagnostics 和 stale job sample，而不是只能读取 raw JSON。

## Acceptance

1. panel 显示 `Runtime Health`。
2. panel 显示 `No runtime health snapshot fields`、`No runtime health errors`。
3. panel 显示 `Stale Jobs`。
4. panel 显示 `No worker stale-job diagnostics sampled`、`No stale jobs sampled`。
5. panel 显示 `Latest Updated`。
6. unified goal verifier 返回：
   - `dashboard_read_models.runtime_health_table=true`
   - `dashboard_read_models.stale_jobs_table=true`
