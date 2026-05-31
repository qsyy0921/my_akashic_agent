# 044 Dashboard Proactive Tick Log Runtime Fallback

Date: 2026-05-31

## 背景

Phase 8.100 已把 proactive tick start / finish / step 审计状态迁移到 Go
`ProactiveStateService`，Python 继续写 SQLite mirror 以兼容现有 dashboard。
如果 SQLite mirror 为空、损坏或尚未写入，dashboard proactive tick 页面仍只能看到空结果，
而 Go runtime 已经持有权威 tick log。

本切片补齐 dashboard 所需的 Go tick log 查询能力，并在 Python dashboard API 增加
SQLite-first / Go-fallback 读路径。这样不破坏旧页面，也让 Go-owned runtime state
可以承担缺失 mirror 时的只读诊断。

## Go / Python 边界

Go 负责：

- tick log / tick step 的确定性状态持久化。
- dashboard 所需的 tick log 过滤、分页、排序、时间区间查询。
- tick detail 和 step detail 的只读 API。

Python 负责：

- dashboard HTTP API 的兼容路由。
- SQLite mirror 的历史兼容读取。
- Go runtime 不可用或无结果时的 fallback 策略。
- 不参与 tick log 的业务推理、LLM、工具执行或平台发送。

## 范围

Go `GET /v1/proactive/tick-logs` 增强查询参数：

- `limit`
- `offset`
- `session_key`
- `terminal_action`
- `gate_exit`
- `flow`
- `started_from`
- `started_to`
- `sort_by`
- `sort_order`

Python dashboard API：

- `/api/dashboard/proactive/tick_logs` 保持 SQLite 优先；SQLite 无结果时读取 Go。
- `/api/dashboard/proactive/tick_logs/{tick_id}` SQLite 未命中时读取 Go。
- `/api/dashboard/proactive/tick_logs/{tick_id}/steps` SQLite 未命中时读取 Go tick 与 step。
- runtime 不可用、超时、返回异常、JSON 不合法时继续返回 SQLite 结果或原 404。

## 不做

- 不切换 proactive overview 全量统计为 Go 权威。overview 仍以 SQLite mirror 为主，后续 live
  验证后再单独切换。
- 不迁移 `semantic_items`，它仍属于语义候选/Embedding/RAG 实验态。
- 不改变 proactive loop、gate、LLM、tool、dispatch 行为。

## 验收

- Go handler/store/service tests 覆盖 offset、sort、started_at 区间过滤。
- Python dashboard tests 覆盖 SQLite 无结果时从 Go runtime 读取 tick list/detail/steps。
- Runtime 不可用时现有 SQLite dashboard 测试保持通过。
- `go test ./app/service ./infrastructure/proactivestate ./infrastructure/memory ./trigger/http` 通过。
- `uv run pytest tests/test_dashboard_api.py -q` 通过。
