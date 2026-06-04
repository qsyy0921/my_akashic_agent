# ATDD - Dashboard Media Asset Retention Table

## Scenario

当 operator 打开 runtime overview dashboard 并查看 `media_asset_retention` detail
时，应能直接看到 retention totals、recent assets 和 notes，而不是只能读取 raw
JSON。

## Acceptance

1. panel 显示 `Media Asset Retention`。
2. panel 显示 `Retention Class`、`Cleanup Due`。
3. panel 显示 `No media asset retention assets sampled` 的空态字面串。
4. unified goal verifier 返回：
   - `dashboard_read_models.media_asset_retention_table=true`
