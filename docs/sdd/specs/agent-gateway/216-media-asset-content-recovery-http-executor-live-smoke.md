# 216 Media Asset Content Recovery HTTP Executor Live Smoke

最后更新：2026-06-03

## 问题

当前仓库已经有 Go-owned `media_asset_content` recovery control plane：

- recovery plan
- approval-bound preflight
- HTTP/HTTPS cache recovery executor
- control-mutation audit
- runtime overview / dashboard read-model

但在本轮之前，repo-owned current-turn 证据仍主要停留在 boundary verifier，
不能直接证明：

1. 缺 approval 时 preflight 会阻断且不会下载远端内容；
2. dry-run 不会写 cache；
3. 正式 recovery 会把 HTTP/HTTPS 内容写进本地 cache、更新 registry；
4. `/v1/media-assets/{asset_id}/content` 会读取恢复后的本地内容；
5. applied control-mutation audit 会真实落账。

## 目标

新增一个 repo-owned isolated live smoke：

- 启动隔离 temp `agent-runtime`
- 启动本地临时 HTTP source
- 注册一个远端 `image/png` media asset
- 验证 preflight / approval / dry-run / live recovery / content readback /
  control audit 整体闭环

## 非目标

- 不验证 QQ/Telegram 私有源会话态抓取
- 不验证后台自动重试
- 不触发 OCR / VLM / RAG / Python AI enrichment
- 不修改当前长期运行 runtime 的 content roots、worker 或平台发送配置

## 设计

### 入口

- Python verifier:
  `scripts/verify_media_asset_content_recovery_live_smoke.py`
- PowerShell wrapper:
  `scripts/verify-media-asset-content-recovery-live-smoke.ps1`

### smoke 流程

1. 拉起 temp HTTP source，固定返回 `image/png` bytes
2. 拉起隔离 temp `agent-runtime`
3. 显式设置：
   - `AKASHIC_RUNTIME_STATE_DIR`
   - `AKASHIC_MEDIA_ASSET_ROOTS=<temp-cache-root>`
   - `AKASHIC_MEDIA_CONTENT_RECOVERY_CACHE_ROOT=<temp-cache-root>`
4. `POST /v1/media-assets` 注册一个 HTTP asset
5. `GET /v1/media-assets/content-recovery/preflight`，无 approval 时应阻断
6. `POST /v1/operator-approvals`
7. 再次 preflight，应变成 ready
8. `POST /v1/media-assets/content-recovery` dry-run
9. `POST /v1/media-assets/content-recovery` 正式 recovery
10. 验证 registry、content endpoint、control-mutations ledger

### 必须验证的不变式

1. preflight 无 approval 时 `reason=missing_approval_id`
2. dry-run 不创建 cache 文件，不触发远端下载
3. live recovery 只在正式执行时下载一次远端 HTTP 内容
4. recovery 后 asset metadata 必须包含：
   - `local_path`
   - `recovered_from_url`
   - `recovery_scope=download_to_local_cache`
   - `recovery_approval_id`
   - `recovery_mutation_id`
5. content endpoint 返回恢复后的本地 bytes
6. control mutation audit 出现 `media_asset_content/recover_content`
   applied record

### Unified verifier 接入

`scripts/verify-go-migration-goal.ps1` 必须接入：

- 顶层 `media_recovery_executor_smoke`
- `residual_classification.media_recovery_http_https_executor_smoke`
- `migration_residuals.media_recovery_private_source_executor`
  在 smoke 通过时升级为
  `go_http_https_executor_live_verified_private_source_executor_incomplete`

## 验收

1. `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-media-asset-content-recovery-live-smoke.ps1`
   返回 `conclusion.status=live_verified`
2. unified verifier 当前 turn 输出：
   - `media_recovery_executor_smoke.conclusion.status=live_verified`
   - `migration_residuals.media_recovery_private_source_executor.http_https_executor_smoke=go_http_https_media_recovery_executor_live_verified`
3. 当前 open gap 仍明确留在私有源抓取、后台重拉和恢复后 AI enrichment
