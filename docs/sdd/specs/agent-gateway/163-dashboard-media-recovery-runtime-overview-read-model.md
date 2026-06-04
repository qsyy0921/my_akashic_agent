# Spec 163: dashboard media recovery runtime-overview read model

## Status

Accepted for the current iteration.

## Context

`SPEC-162` 已经把 `OI-005` 收敛成 repo-owned live verifier，但本轮 live 结果
暴露出两个更具体的问题：

1. Go `/v1/runtime-overview` 已有 top-level
   `media_asset_content_recovery` detail 和对应 summary keys；
2. dashboard `/api/dashboard/runtime-overview` 却没有稳定透出
   `media_asset_content_recovery` card/detail；
3. 更深一层地看，dashboard reader 默认 `0.5s` timeout 对
   `/v1/runtime-overview` 不够，读超时后会静默退回 fallback 路径，于是
   `media_asset_content_recovery` 被降级成 `unknown/muted`。

这使得 dashboard 上的 media recovery 结论和 Go runtime 不一致，也让
`verify-media-recovery-boundary.ps1` 只能把 read-model gap 继续记为 open。

## Decision

本轮修复 dashboard runtime-overview read model：

1. `plugins/runtime_overview/dashboard.py` 在 normalize Go runtime overview 时：
   - 透传 top-level `media_asset_content_recovery` detail；
   - 补齐 `media_asset_content_recovery_*` summary 默认值；
   - 若 Go runtime `cards` 中没有该 card，则基于 summary/detail 合成
     `Media Content Recovery` card。
2. dashboard reader fallback 路径也必须返回稳定的
   `media_asset_content_recovery` top-level field 和 muted/unknown card，
   避免字段直接消失。
3. 对 `/v1/runtime-overview` 的 dashboard reader timeout 提升到现实可用范围，
   避免本机 live runtime 因 read timeout 被错误降级到 fallback。

## Out of Scope

- 修改 Go media recovery executor 的 source-scheme 支持；
- 新增 HTTP/HTTPS live recovery candidate；
- 实现 QQ/Telegram 私有源 session/token/cookie fetch；
- 恢复后自动触发 OCR/VLM/RAG/AI enrichment；
- 修改 QQ 群发送策略或恢复 QQ 群发。

## Acceptance

- `/api/dashboard/runtime-overview` 在当前 live runtime 上包含：
  - `media_asset_content_recovery` top-level detail；
  - `cards[].id == "media_asset_content_recovery"`；
  - detail `reason=media_asset_content_recovery_audit_ready`。
- `scripts/verify-media-recovery-boundary.ps1` 当前返回：
  - `dashboard.runtime_overview_has_media_content_recovery_card=true`
  - `dashboard.runtime_overview_has_media_content_recovery_detail=true`
  - `checks.dashboard_runtime_overview_missing_media_recovery_detail=false`
- `scripts/verify-go-migration-goal.ps1` 继续保持 open blockers 不新增 dashboard
  media recovery read-model gap。
- 对应测试、ATDD、TDD、DONE/LIVE_CHECKS/OPEN_ISSUES/REMAINING/TODO 和 spec
  index 同步更新。
