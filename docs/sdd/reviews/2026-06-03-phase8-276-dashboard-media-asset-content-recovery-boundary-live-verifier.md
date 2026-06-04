# Review: Dashboard Media Asset Content Recovery Boundary Live Verifier

日期：2026-06-03

## 变更

- 新增 repo-owned live verifier：
  - `scripts/verify_dashboard_media_asset_content_recovery_boundary.py`
  - `scripts/verify-dashboard-media-asset-content-recovery-boundary.ps1`
- 修复 dashboard fallback：
  - `plugins/runtime_overview/dashboard.py`
- 接入 unified goal verifier：
  - `scripts/verify-go-migration-goal.ps1`

## 结果

- dashboard fallback 现在会从 `control-mutations` audit ledger 合成
  `media_asset_content_recovery` summary/card/detail
- live dashboard parity verifier 当前返回：
  - `status=live_verified`
  - `category=dashboard_media_asset_content_recovery_boundary_live_verified`
- unified artifact 当前返回：
  - `dashboard_read_models.media_asset_content_recovery_table=true`
  - `checks.dashboard_read_models.media_asset_content_recovery_table=true`

## 剩余边界

这次收掉的是 dashboard/runtime 的只读 parity，不是 private-source executor。

剩余 media recovery gap 仍然是：

- `qq_private_source_session_fetch`
- `telegram_private_source_session_fetch`
- `background_repull`
- `post_recovery_ai_enrichment`
