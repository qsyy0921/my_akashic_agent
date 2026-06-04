# 220. Dashboard Media Asset Content Boundary Live Verifier

## 背景

Go runtime 已经提供 `media_asset_content` 只读诊断：

- `/v1/media-assets/content-diagnostics`
- `/v1/runtime-overview` 中的 `media_asset_content_*` summary
- `Media Asset Content` card/detail
- per-asset `content_recovery_plan_endpoint`
- per-asset `content_recovery_preflight_endpoint`

但 dashboard `/api/dashboard/runtime-overview` 在 runtime overview 聚合超时后会走
fallback。此前 fallback 没有补 `media_asset_content`，导致 current-turn live
runtime 已有 detail，而 dashboard 仍可能缺：

- top-level `media_asset_content_diagnostics`
- `Media Asset Content` card
- per-asset recovery/preflight endpoint

这会让 read-model evidence 失真，也会让 operator 在 dashboard 中看不到
content recovery preflight 链接。

## 目标

补齐 repo-owned live verifier，并让 dashboard fallback 在只读前提下保留
`media_asset_content` 的 runtime parity。

## 非目标

本次不做以下事情：

- 不执行 content recovery
- 不创建 approval / control mutation
- 不下载、恢复、缓存 media content
- 不触发 OCR / VLM / RAG / AI enrichment
- 不修改 private-source executor 边界

## 设计

### 1. dashboard fallback 读取 content diagnostics

`plugins/runtime_overview/dashboard.py` 的 fallback 路径新增读取：

- `/v1/media-assets/content-diagnostics`

并把结果规范化到 dashboard runtime-overview payload 中：

- `summary.media_asset_content_*`
- top-level `media_asset_content_diagnostics`
- `Media Asset Content` card/detail

### 2. 规范化透传 recovery / preflight endpoint

`_normalize_media_asset_content_item()` 必须保留：

- `content_recovery_plan_endpoint`
- `content_recovery_preflight_endpoint`

如果上游 item 缺字段，则按 `asset_id` 合成 deterministic fallback URL。

### 3. 新增 repo-owned live verifier

新增：

- `scripts/verify_dashboard_media_asset_content_boundary.py`
- `scripts/verify-dashboard-media-asset-content-boundary.ps1`

verifier 直接对比：

- Go `/v1/runtime-overview`
- dashboard `/api/dashboard/runtime-overview`

校验内容：

- `media_asset_content_*` summary parity
- `Media Asset Content` card parity
- `media_asset_content_diagnostics` detail parity
- sampled asset parity
- dashboard `/api/dashboard/media-assets/content-recovery/preflight`
  与 Go `/v1/media-assets/content-recovery/preflight` 的 proxy parity

同时要求该边界保持只读：

- detail `side_effect=none`
- preflight `side_effect=none`

### 4. 接入 unified goal verifier

`scripts/verify-go-migration-goal.ps1` 新增并固化：

- `dashboard_media_asset_content_boundary`
- `residual_classification.dashboard_media_asset_content_boundary`
- `dashboard_read_models.media_asset_content_table`
- `checks.dashboard_read_models.media_asset_content_table`

并把这条证据纳入 `dashboard_fallback_category=live_verified_runtime_read_models`
的 current-turn gate。

## 验收

满足以下条件即视为完成：

1. `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-dashboard-media-asset-content-boundary.ps1`
   返回 `live_verified`
2. dashboard `/api/dashboard/runtime-overview` 暴露 `media_asset_content_diagnostics`
   与 `Media Asset Content` card
3. sampled asset 含 `content_recovery_plan_endpoint` 与
   `content_recovery_preflight_endpoint`
4. `.codex-goal-verifier.json` 中：
   - `dashboard_read_models.media_asset_content_table=true`
   - `checks.dashboard_read_models.media_asset_content_table=true`
   - `residual_classification.dashboard_media_asset_content_boundary.status=live_verified`
