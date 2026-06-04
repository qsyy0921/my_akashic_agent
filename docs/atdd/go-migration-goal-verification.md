# Go Migration Goal Verification ATDD

Date: 2026-06-02

## Goal

提供一个 repo-owned 统一验证入口，用当前 live runtime 结果支持每轮 goal 更新。

## Acceptance Scenarios

### 1. 默认只读验证

前置条件：

- `agent-runtime` 在 `127.0.0.1:8780` 运行。

步骤：

1. 运行 `.\scripts\verify-go-migration-goal.ps1`

期望结果：

- 脚本返回单个 JSON 文档。
- JSON 包含：
  - `qq_outbox_cutover`
  - `qq_rich_media`
  - `telegram_backend`
  - `knowledge_planner`
  - `residual_classification`
  - `checks.open_blockers`
- 默认不会发送 QQ/Telegram 消息。

### 2. Telegram blocker 分类

前置条件：

- 当前环境可能没有 Telegram token。

步骤：

1. 运行 `.\scripts\verify-go-migration-goal.ps1`

期望结果：

- 当 token 缺失时，`telegram_backend.checks.backend_state=token_missing`
- 不伪造 `getMe` 成功或 Telegram receiver 就绪。

### 3. Knowledge planner 健康复核

前置条件：

- Go knowledge planner 已启用。

步骤：

1. 运行 `.\scripts\verify-go-migration-goal.ps1`

期望结果：

- `knowledge_planner.conclusion=go_admission_owner_healthy`
- 最近 `group_memory_extract` jobs 仍来自 `agent-runtime-knowledge-job-planner`

### 4. 可选 rich-media probe

前置条件：

- NapCat WebSocket 可访问。

步骤：

1. 运行 `.\scripts\verify-go-migration-goal.ps1 -IncludeNativeRichMediaProbe -RichMediaProbeGroupId 3219982`

期望结果：

- JSON 中 `qq_rich_media.native_probe_enabled=true`
- `qq_rich_media.native_probe` 包含原生 NapCat probe 结果
- 该 probe 结果可用于刷新当前 rich-media blocker，而不是复述旧文档
