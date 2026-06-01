# Runtime Overview Media Retention

日期：2026-06-01

## 背景

`/v1/media-assets/retention-diagnostics` 已经提供 Go-owned 媒体资产保留策略只读诊断，但 runtime overview 和 Python dashboard 还不能直接展示 cleanup due 风险。Operator 需要在总览页看到是否存在过期附件候选，而不是单独记住新 endpoint。

## 目标

- Go `/v1/runtime-overview` 聚合 media asset retention diagnostics。
- Summary 暴露：
  - `media_asset_retention_assets`
  - `media_asset_retention_cleanup_due`
  - `media_asset_retention_permanent`
  - `media_asset_retention_default`
  - `media_asset_retention_ephemeral`
  - `media_asset_retention_unknown`
- Runtime card 新增 `media_asset_retention`，value 为 cleanup_due/assets，status 只做 advisory。
- Python dashboard aggregate normalization 和 fallback 同步支持 `media_asset_retention_diagnostics`。

## 非目标

- 不自动删除媒体 registry 或文件。
- 不新增 cleanup mutation。
- 不改变 content diagnostics、safe root、OCR/VLM、文件解析逻辑。

## Go / Python 边界

- Go：媒体资产 registry、retention diagnostics、runtime overview 稳定控制面。
- Python：dashboard 展示适配和 AI pipeline；不执行 retention cleanup。

## 验证

- `go test ./app/service`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q`
- `go test ./...`
- `git diff --check`
