# Phase 8.212 Review - Goal Verification Hardening

Date: 2026-06-02

## Summary

本轮修掉了两个真实运行问题，并把 Telegram blocker 的证据链补完整：

1. OneBot WebSocket 失败响应里的结构化 `status` 不再打崩 Go outbox worker
2. 本地 launcher 现在可以显式启用 outbox worker
3. Telegram verifier 现在能证明 repo 已声明 Telegram channel，但 token 仍未被注入

## What Changed

- `onebotResponse.Status` 改成容错字段类型，接受非字符串 JSON
- `start-agent-runtime.ps1` 新增 `-EnableOutboxDeliveryWorker`
- `verify-telegram-backend.ps1` 新增 config / `.env` / `~/.codex/.env` 证据输出
- `verify-go-migration-goal.ps1` 继续复用更新后的 Telegram verifier

## Verification

已运行：

```powershell
C:\Users\10495\AppData\Local\Programs\Go\bin\go.exe test ./infrastructure/onebotdelivery
.\scripts\verify-telegram-backend.ps1 -RepoRoot E:\agent\my-akashic_agent
.\scripts\verify-go-outbox-scope-live.ps1
.\scripts\verify-go-migration-goal.ps1 -IncludeOutboxScopeSmoke -IncludeNativeRichMediaProbe -RichMediaProbeGroupId 3219982
uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q
```

关键 live 结果：

- `verify-telegram-backend.ps1` 现在明确显示：
  - `config_declares_telegram_channel=true`
  - `config_token_line=${TELEGRAM_BOT_TOKEN}`
  - `config_uses_env_placeholder=true`
  - `repo_env_has_telegram_key=false`
  - `codex_env_has_telegram_key=false`
  - `backend_state=token_missing`
- 重新启动后的 runtime 仍保持：
  - `outbox_delivery_worker.enabled=true`
  - `knowledge_job_planner.enabled=true`
- `verify-go-outbox-scope-live.ps1` 重新确认：
  - first-account private file -> succeeded
  - second-account group file -> succeeded
  - first-account group file -> queued
  - first-account private image -> queued
- 统一 goal verifier 最终恢复到：
  - `outbox_conclusion=partial_go_default_owner_live_verified`
  - `open_blockers` 不再包含 outbox scope false negative

## Risks / Follow-up

- QQ `image` 和第一账号 `group file` 仍是外部平台/session blocker。
- Telegram 仍缺 token；当前只是把“为什么没通”证明得更硬了。
