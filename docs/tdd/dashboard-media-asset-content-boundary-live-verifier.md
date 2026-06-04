# Dashboard Media Asset Content Boundary Live Verifier TDD

## 新增测试

- `tests/test_runtime_overview_dashboard_plugin.py`
  - fallback 路径读取 `/v1/media-assets/content-diagnostics`
  - fallback summary 暴露 `media_asset_content_*`
  - fallback card 暴露 `media_asset_content`
  - sampled item 保留 `content_recovery_plan_endpoint`
  - sampled item 保留 `content_recovery_preflight_endpoint`

- `tests/test_verify_dashboard_media_asset_content_boundary.py`
  - happy path parity
  - preflight proxy drift detection
  - verifier status/category 分支

- `tests/test_verify_go_migration_goal_script.py`
  - unified verifier wiring
  - `dashboard_read_models.media_asset_content_table`
  - `dashboard_media_asset_content_boundary` top-level / residual wiring

## 回归检查

- runtime overview dashboard 其它已结构化 drilldown 不回退
- `dashboard_fallback_category` 只有在所有 tracked read-model 证据齐全时才保持
  `live_verified_runtime_read_models`
