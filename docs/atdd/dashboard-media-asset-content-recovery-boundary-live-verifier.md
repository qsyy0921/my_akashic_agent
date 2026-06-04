# ATDD: Dashboard Media Asset Content Recovery Boundary Live Verifier

## 前置条件

- Go runtime 运行在 `http://127.0.0.1:8780`
- Python dashboard 运行在 `http://127.0.0.1:2236`

## 验收步骤

1. 运行：
   - `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-dashboard-media-asset-content-recovery-boundary.ps1`
2. 期望结果：
   - `conclusion.status=live_verified`
   - `conclusion.category=dashboard_media_asset_content_recovery_boundary_live_verified`
3. 运行：
   - `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
4. 期望 `.codex-goal-verifier.json` 中：
   - `dashboard_read_models.media_asset_content_recovery_table=true`
   - `checks.dashboard_read_models.media_asset_content_recovery_table=true`
   - `residual_classification.dashboard_media_asset_content_recovery_boundary.category=dashboard_media_asset_content_recovery_boundary_live_verified`

## 副作用约束

整个过程只允许读取：

- `/v1/runtime-overview`
- `/api/dashboard/runtime-overview`
- fallback 所需只读 audit state

不得：

- 调用 recovery executor
- 创建 approval / mutation
- 下载 / 缓存媒体内容
- 触发 OCR / VLM / RAG / AI

