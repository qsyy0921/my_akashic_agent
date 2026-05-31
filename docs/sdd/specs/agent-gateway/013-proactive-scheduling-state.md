# 013 Proactive Scheduling State

Date: 2026-05-30

## 背景

`proactive_v2.state.ProactiveStateStore` 当前把主动推送的确定性状态保存在
Python SQLite 中。这里面既有 AI 过程日志，也有适合 Go 承担的后端基础设施状态。

本切片只迁移主动推送调度所需的稳定状态：

- delivery 去重记录：`session_key + delivery_key -> sent_at`
- delivery 窗口计数：用于主动推送冷却和疲劳因子
- context-only 发送记录：用于 fallback 触达的间隔和每日上限
- drift 运行时间标记：用于 drift 最小运行间隔

Python 仍负责 prompt、LLM 决策、候选内容分类、发送内容生成和 tick 日志。

## 目标

1. 在 Go `agent-runtime` 中增加 proactive scheduling state 领域模型。
2. 提供本地 JSON 持久化存储，支持开发机重启后继续判断冷却状态。
3. 暴露 HTTP API，供 Python compatibility store 后续替换 SQLite 调用。
4. 不改变当前 proactive loop 的默认行为；未启用 runtime 接入时继续走 Python SQLite。

## 非目标

- 不迁移 proactive tick log / tick step log。
- 不迁移 semantic items。source seen items 和 rejection cooldown 已在后续
  `033-proactive-seen-and-rejection-state.md` 迁移。
- 不改变 LLM 判断策略。
- 不直接接管 Telegram/QQ 发送。

## 分层设计

```text
trigger/http
  -> app/port/in/ProactiveStateManager
  -> app/service/ProactiveStateService
  -> app/port/out/ProactiveStateRepository
  -> infrastructure/proactivestate{json}
  -> domain/model
```

## API

| Route | Method | 用途 |
| --- | --- | --- |
| `/v1/proactive/deliveries` | `POST` | 记录一次主动发送 |
| `/v1/proactive/deliveries` | `GET` | 查询最近发送记录 |
| `/v1/proactive/deliveries/duplicate` | `GET` | 判断窗口内是否重复发送 |
| `/v1/proactive/deliveries/count` | `GET` | 统计窗口内发送次数 |
| `/v1/proactive/context-only` | `POST` | 记录一次 context-only 发送 |
| `/v1/proactive/context-only/last` | `GET` | 查询最近一次 context-only |
| `/v1/proactive/context-only/count` | `GET` | 统计窗口内 context-only 次数 |
| `/v1/proactive/drift-runs` | `POST` | 记录 drift 运行时间 |
| `/v1/proactive/drift-runs/last` | `GET` | 查询最近一次 drift 运行时间 |

## 存储

默认可使用内存 store；生产/本地长期运行通过：

```text
AKASHIC_PROACTIVE_STATE_DSN=E:\agent\akashic\.akashic-workspace\runtime\proactive-state.json
```

也支持兼容变量：

```text
AKASHIC_PROACTIVE_STATE_PATH=...
```

## 兼容策略

第一步只增加 Go-owned API 和持久化。下一步 Python 可增加
`AgentRuntimeProactiveStateStore`，在 `agent_runtime.enabled=true` 且 base URL
存在时委托给 Go API；失败时回退到 SQLite，避免主动推送链路中断。

## 验收

- Go 单元测试覆盖记录、重复判断、窗口计数、last marker。
- HTTP 测试覆盖所有核心路由。
- JSON store 重启后保留 delivery/context/drift 状态。
- `go test ./...` 通过。
