# 155. Goal Verification Hardening

Date: 2026-06-02

## Status

Accepted

## Context

在统一 goal 验证和 outbox scope live smoke 落地后，又暴露出两个真实问题：

1. NapCat 某些 WebSocket 失败响应会把 `status` 返回成对象，而不是字符串。
   当前 `onebotResponse.Status string` 会在 JSON 解析阶段直接报错，
   让 Go outbox worker 把真实平台错误误记成内部解析失败。
2. `scripts/start-agent-runtime.ps1` 无法显式启用 outbox worker，
   只能依赖当前 shell 是否已带 `AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED`。
   这会让本地 runtime 重启后偏离预期 scope。
3. Telegram 当前 blocker 需要更强证据：
   不只是“环境里没 token”，还要证明 repo 配置声明了 Telegram channel，
   但 token 仍停留在 `${TELEGRAM_BOT_TOKEN}` 占位符，且 repo `.env` /
   `~/.codex/.env` 里也没有对应键。

## Decision

本轮做三件事：

1. `onebotResponse.Status` 改成容错字段类型，接受：
   - string
   - object
   - null
   并保证非字符串 status 不会阻断后续 `retcode/message` 错误分类。
2. `scripts/start-agent-runtime.ps1` 新增 `-EnableOutboxDeliveryWorker`，
   允许本地 bring-up 显式启用 Go local outbox worker。
3. `scripts/verify-telegram-backend.ps1` 增强 config 证据输出：
   - `config.toml` 是否存在
   - 是否声明 `[channels.telegram]`
   - token 行是否仍是 env placeholder
   - repo `.env` 是否存在且是否含 Telegram key
   - `~/.codex/.env` 是否存在且是否含 Telegram key

## Consequences

正面：

- Go outbox worker 不再因 OneBot `status` 字段结构变化而误判失败。
- 本地 runtime 重启可以稳定恢复目标 outbox scope。
- Telegram blocker 现在可以明确表达为：
  “通道已声明，但运行时和已检查配置来源里都没有 token 注入”。

负面：

- 这不会解决 QQ rich-media 的平台 blocker。
- 这不会凭空生成 Telegram token；只能把证据链做完整。

## Verification

最小验证：

```powershell
C:\Users\10495\AppData\Local\Programs\Go\bin\go.exe test ./infrastructure/onebotdelivery
.\scripts\verify-telegram-backend.ps1 -RepoRoot E:\agent\my-akashic_agent
.\scripts\start-agent-runtime.ps1 -EnableOutboxDeliveryWorker -OutboxDeliveryAllowedKinds 'text' -OutboxDeliveryAllowedKindsByAccount '2365524513=text|file' -OutboxDeliveryAllowedKindsByAccountConversationType '1049511700/private=text|file' -EnableKnowledgeJobPlanner
.\scripts\verify-go-outbox-scope-live.ps1
.\scripts\verify-go-migration-goal.ps1 -IncludeOutboxScopeSmoke -IncludeNativeRichMediaProbe -RichMediaProbeGroupId 3219982
```

期望结果：

- OneBot 包测试通过
- Telegram verifier 显示 `config_declares_telegram_channel=true`、
  `config_uses_env_placeholder=true`、`local_env_token_present=false`
- 重启后的 runtime 仍带正确 outbox worker scope
- unified goal verifier 最终保持：
  - `outbox_conclusion=partial_go_default_owner_live_verified`
  - `telegram_backend_state=token_missing`
