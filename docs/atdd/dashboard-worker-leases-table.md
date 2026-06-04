# ATDD - Dashboard Worker Leases Table

## Scenario

当 operator 打开 runtime overview dashboard 并查看 `worker_leases` detail 时，应能
直接看到 lease diagnostics 的结构化表格，而不是只能读取 raw JSON。

## Acceptance

1. panel 显示 `Worker Leases`。
2. panel 显示 `Checkpoint Prefix` 和 `Stale Leases` 列。
3. 没有样本时显示 `No worker lease diagnostics sampled`。
4. unified goal verifier 返回 `dashboard_read_models.worker_leases_table=true`。
