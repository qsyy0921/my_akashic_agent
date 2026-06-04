# Goal Verification Hardening ATDD

Date: 2026-06-02

## Goal

让本地 goal 验证在真实运行态下更稳定，并让 Telegram blocker 与 Go outbox scope 的证据更完整。

## Acceptance Scenarios

### 1. OneBot 结构化 status 不再打崩 outbox worker

前置条件：

- Go runtime 已使用更新后的 `onebotdelivery` 代码启动。

步骤：

1. 运行 `.\scripts\verify-go-outbox-scope-live.ps1`

期望结果：

- 不会再因为 `status` 字段是对象而出现 JSON unmarshal 错误
- 当前已放开的 `file` 路由可以正常完成或返回真实平台错误

### 2. Launcher 可显式启用 outbox worker

前置条件：

- `agent-runtime` 未占用 8780 端口

步骤：

1. 运行：
   `.\scripts\start-agent-runtime.ps1 -EnableOutboxDeliveryWorker ...`

期望结果：

- `/v1/runtime-workers` 中 `outbox_delivery_worker.enabled=true`
- 不依赖外部 shell 是否已有 `AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED`

### 3. Telegram blocker 具备配置证据

前置条件：

- repo 根目录存在 `config.toml`

步骤：

1. 运行 `.\scripts\verify-telegram-backend.ps1 -RepoRoot E:\agent\my-akashic_agent`

期望结果：

- 返回：
  - `config_toml_exists=true`
  - `config_declares_telegram_channel=true`
  - `config_uses_env_placeholder=true`
  - `local_env_token_present=false`
  - `backend_state=token_missing`
