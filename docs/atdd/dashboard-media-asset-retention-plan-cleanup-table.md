# ATDD - Dashboard Media Asset Retention Plan Cleanup Table

## Scenario

当 operator 打开 runtime overview dashboard 并查看
`media_asset_retention_plan` 与 `media_asset_retention_cleanup` detail 时，应能直接看
到 retention candidate、required steps、cleanup totals 与 notes，而不是只能读取
raw JSON。

## Acceptance

1. panel 显示 `Media Asset Retention Plan`。
2. panel 显示 `Required Steps`、`Verification Steps`、`Rollback Steps`。
3. panel 显示 `Media Asset Retention Cleanup`。
4. panel 显示 `Candidates`、`Rolled Back`。
5. unified goal verifier 返回：
   - `dashboard_read_models.media_asset_retention_plan_table=true`
   - `dashboard_read_models.media_asset_retention_cleanup_table=true`
