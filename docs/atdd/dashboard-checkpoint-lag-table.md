# ATDD - Dashboard Checkpoint Lag Table

## Scenario

当 operator 打开 runtime overview dashboard 并查看 `checkpoint_lag` detail 时，应能
直接看到 lagged checkpoints 和 freshness 字段，而不是只能读取 raw JSON。

## Acceptance

1. panel 显示 `Checkpoint Lag`。
2. panel 显示 `No lagged checkpoints sampled`。
3. panel 显示 `Lag Messages`、`Checkpoint Seq`。
4. unified goal verifier 返回：
   - `dashboard_read_models.checkpoint_lag_table=true`
