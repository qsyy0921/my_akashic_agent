# 043 Proactive Tick Log State

Date: 2026-05-31

## 背景

Proactive loop 的 tick start / finish / step log 目前保存在 Python
`proactive.db` 的 `tick_log` 和 `tick_step_log` 表中。它们不是 LLM 推理能力，
而是运行审计、诊断和 dashboard 回放状态，适合交给 Go runtime control plane。

本切片只迁移确定性的审计写入和只读查询。Python 继续执行 gate、fetch、LLM judge、
tool loop、message dispatch、ACK 和 SQLite mirror，dashboard 也暂时继续读取 SQLite，
避免一次性改动前端路径。

## 范围

新增到 Go：

- `ProactiveTickLog` domain model：tick id、session、start/finish time、gate exit、
  terminal action、计数、引用 id、drift 标记和最终消息摘要。
- `ProactiveTickStepLog` domain model：tick id、step index、phase、tool 名称、
  tool call id、tool args、结果文本和 step 后状态。
- file-backed `proactive-state.json` 和 in-memory store 的 tick log / step log 支持。
- HTTP 写入口：
  - `POST /v1/proactive/tick-logs/start`
  - `POST /v1/proactive/tick-logs/finish`
  - `POST /v1/proactive/tick-steps`
- HTTP 只读查询：
  - `GET /v1/proactive/tick-logs`
  - `GET /v1/proactive/tick-logs/{tick_id}`
  - `GET /v1/proactive/tick-logs/{tick_id}/steps`

接入到 Python：

- `AgentRuntimeProactiveStateStore.record_tick_log_start()` 先写 Go，再写 SQLite mirror。
- `record_tick_log_finish()` 先写 Go，再写 SQLite mirror。
- `record_tick_step_log()` 先写 Go，再写 SQLite mirror。
- Go runtime 不可用时保留旧 SQLite fallback 行为。

暂不迁移：

- proactive gate / LLM / tool / dispatch 行为。
- `semantic_items`，因为它更接近语义候选/RAG 实验状态。
- dashboard proactive 页面读路径。等 live 验证 Go tick log 稳定后，再单独切换读路径或做 fallback。

## 分层

```text
trigger/http
  -> app/port/in/ProactiveStateManager
  -> app/service/ProactiveStateService
  -> app/port/out/ProactiveStateRepository
  -> domain/model/ProactiveTickLog + ProactiveTickStepLog
  -> infrastructure/proactivestate + infrastructure/memory
```

## 规则

- `tick_id` 必须非空。
- `session_key` 允许为空，以兼容 `no_target` gate exit。
- start/finish 允许幂等 upsert 同一个 `tick_id`。
- step log 为 append-only，按 `tick_id + step_index + insertion order` 查询。
- Go store 保留最近 500 条 tick log 和 5000 条 step log，避免本地 JSON 无限增长。
- 写接口返回 `side_effect=runtime_state_write`；查询接口返回 `side_effect=none`。
- Python 必须继续写 SQLite mirror，保证现有 dashboard 不破坏。

## 验收

- Go service/store/http tests 覆盖 start、finish、step、list、detail、steps 和文件重载。
- Python bridge tests 覆盖 tick log 三个写入口优先调用 Go，且 SQLite mirror 仍写入。
- Runtime 不可用时 Python 旧 SQLite fallback 行为保持。
- `go test ./domain/model ./app/service ./infrastructure/proactivestate ./trigger/http` 通过。
- `uv run pytest tests/test_agent_runtime_proactive_state.py` 通过。
