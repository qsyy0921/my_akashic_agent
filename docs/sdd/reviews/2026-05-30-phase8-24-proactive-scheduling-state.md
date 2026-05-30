# Phase 8.24 Proactive Scheduling State Review

Date: 2026-05-30

## 设计结论

本切片适合进入 Go：delivery 去重、窗口计数、context-only 节流和 drift
间隔判断都是确定性状态，不依赖模型推理，且需要可观测、可持久化和可回放。

## 边界

- Go 负责 proactive scheduling state 的存储、查询和窗口判断。
- Python 继续负责 proactive prompt、候选分类、去重语义判断、生成和发送编排。
- 当前切片不替换 Python SQLite 默认路径，降低回归风险。

## 风险

1. JSON store 仍是本地开发级别，不适合作为多进程高并发队列后端。
2. Python compatibility store 尚未接入，因此这一步是 runtime 能力补齐，不是完整切流。
3. 时间窗口由调用方传入 `now` 或由服务端取当前时间，跨语言测试需要固定时间。

## 后续

- 增加 Python `AgentRuntimeProactiveStateStore`。
- 在 `agent_runtime.enabled` 时让 proactive loop 使用 Go state。
- dashboard proactive 面板逐步读 Go API，SQLite 作为 fallback。

## 验证

- `go test ./...` under `services/agent-runtime`
- `go build ./cmd/agent-runtime`
- Live smoke: `POST /v1/proactive/deliveries` returned `OK` after restarting
  the local `agent-runtime` binary with JSON persistence configured.
