# 035 Proactive Background Context Global Mark

Date: 2026-05-31

## 背景

`bg_context_last_main_at` 是 proactive background context 主 topic 最近一次发送时间。它只是一个全局时间戳，用于节流和间隔判断，不涉及 LLM 推理、候选抽取或 prompt 选择。

这类确定性运行时状态应由 Go `agent-runtime` 持久化，Python 只保留 SQLite fallback。

## 范围

迁移到 Go：

- `get_bg_context_last_main_at`
- `mark_bg_context_main_send`

继续留在 Python/SQLite：

- semantic items。
- tick log / tick step log。
- prompt 和主动推送内容决策。

## API

```text
POST /v1/proactive/bg-context/main
GET  /v1/proactive/bg-context/main/last
```

写入请求：

```json
{
  "timestamp": "2026-05-30T12:30:00Z"
}
```

查询响应复用 timestamp view：

```json
{
  "key": "bg_context_last_main_at",
  "timestamp": "2026-05-30T12:30:00Z",
  "found": true
}
```

## 分层

```text
trigger/http
  -> app/port/in/ProactiveStateManager
  -> app/service/ProactiveStateService
  -> app/port/out/ProactiveStateRepository
  -> infrastructure/proactivestate | infrastructure/memory
  -> domain/model.ProactiveGlobalMark
```

## Python Bridge

`AgentRuntimeProactiveStateStore`：

1. mark 时先写 Go，再写 SQLite fallback。
2. get 时读取 Go 和 SQLite 两侧 timestamp，返回较新的时间。
3. Go 不可用时记录 warning 并回退 SQLite。

## 验收

- Go domain/app/store/http tests 覆盖 global mark 写入、读取和持久化。
- Python bridge mock runtime 覆盖 bg-context route 和 fallback。
- 全量 Go test、go vet、build 和 Python targeted test 通过。
