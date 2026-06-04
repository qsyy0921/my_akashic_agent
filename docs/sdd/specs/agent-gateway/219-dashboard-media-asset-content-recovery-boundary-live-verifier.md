# 219. Dashboard Media Asset Content Recovery Boundary Live Verifier

## 背景

`media_asset_content_recovery` 的 Go runtime control-plane 已经具备：

- recovery plan
- approval-bound preflight
- control mutation audit
- HTTP/HTTPS cache recovery executor live smoke

但 dashboard `/api/dashboard/runtime-overview` 在 runtime overview 聚合超时时会走 fallback。这个 fallback 之前会把 `Media Content Recovery` summary/card/detail 退回 `unknown/muted`，导致：

- runtime `/v1/runtime-overview` 已是 `media_asset_content_recovery_audit_ready`
- dashboard `/api/dashboard/runtime-overview` 却显示 `unknown`

这会让 current-turn artifact 把 read-model 证据读成未验证。

## 目标

补齐 repo-owned live verifier，并让 dashboard fallback 在不调用 recovery side effect 的前提下保留 `media_asset_content_recovery` 的只读 parity。

## 非目标

以下不在本次范围内：

- 执行真正的 media content recovery
- 引入 QQ / Telegram private-source fetch executor
- 触发 OCR / VLM / RAG / AI enrichment
- 修改 HTTP/HTTPS recovery executor 语义

## 设计

### 1. 新增 dashboard/runtime parity verifier

新增：

- `scripts/verify_dashboard_media_asset_content_recovery_boundary.py`
- `scripts/verify-dashboard-media-asset-content-recovery-boundary.ps1`

该 verifier 只读取：

- Go `/v1/runtime-overview`
- dashboard `/api/dashboard/runtime-overview`

并比较：

- `media_asset_content_recovery_*` summary
- `Media Content Recovery` card
- top-level `media_asset_content_recovery` detail

同时要求该边界保持只读：

- `side_effect=none`
- 仅暴露 plan / preflight / recovery endpoint 链接
- 不调用 preflight / recovery endpoint

### 2. 修复 dashboard fallback

当 dashboard 无法在预算内直接消费 Go `/v1/runtime-overview` 聚合结果时，fallback 现在额外读取：

- `/v1/control-mutations?target_kind=media_asset_content`

并据此合成 `media_asset_content_recovery` 的：

- summary
- top-level detail
- card value / status

该合成路径只依赖 control mutation audit ledger，不执行 recovery，不创建 approval，不下载内容。

### 3. 接入 unified goal verifier

`scripts/verify-go-migration-goal.ps1` 新增：

- `dashboard_media_asset_content_recovery_boundary`
- `residual_classification.dashboard_media_asset_content_recovery_boundary`
- `dashboard_read_models.media_asset_content_recovery_table`
- `checks.dashboard_read_models.media_asset_content_recovery_table`

同时把这条 live verifier 纳入 `dashboard_fallback_category=live_verified_runtime_read_models` 的判定。

## 验收

满足以下条件即视为完成：

1. `scripts/verify-dashboard-media-asset-content-recovery-boundary.ps1` 返回 `live_verified`
2. dashboard `/api/dashboard/runtime-overview` 的 `media_asset_content_recovery` summary/card/detail 与 Go `/v1/runtime-overview` 一致
3. fallback 模式下仍不退回 `unknown/muted`
4. `.codex-goal-verifier.json` 中
   - `dashboard_read_models.media_asset_content_recovery_table=true`
   - `checks.dashboard_read_models.media_asset_content_recovery_table=true`
   - `residual_classification.dashboard_media_asset_content_recovery_boundary.status=live_verified`

