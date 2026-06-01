# Phase 8.161 Review: Runtime Overview Media Retention

日期：2026-06-01

## 范围

- Go runtime overview 聚合 `MediaAssetRetentionDiagnostics`。
- 新增 summary 字段和 `Media Asset Retention` card。
- Python dashboard aggregate normalization 和 fallback 支持 `media_asset_retention_diagnostics`。
- 更新 SDD TODO / DONE / LIVE_CHECKS。

## 架构检查

- Go 继续负责 media asset registry、retention diagnostics 和 runtime overview 控制面。
- Python dashboard 只做展示适配；OCR/VLM、文件解析和语义抽取仍归 Python AI pipeline。
- 本轮只读，未新增 cleanup mutation、未删除 registry、未删除文件、未改变 content endpoint。

## 行为

- Summary 暴露 `media_asset_retention_assets`、`media_asset_retention_cleanup_due`、`media_asset_retention_permanent`、`media_asset_retention_default`、`media_asset_retention_ephemeral`、`media_asset_retention_unknown`。
- Card value 为 `cleanup_due/assets`。
- `cleanup_due>0` 时 card status 为 `warn`；无资产为 `muted`；无 due 为 `ok`。

## 验证

- `go test ./app/service`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q`
- `go test ./...`
- `git diff --check`

## 风险

- Overview 只是 advisory 视图，不代表 retention cleanup 已执行。
- 如果未来新增真实清理，必须绑定 operator approval、control mutation audit、dry-run/commit 分离和回滚策略。
