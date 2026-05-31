# 034 Proactive Retention Cleanup

Date: 2026-05-31

## 背景

`033-proactive-seen-and-rejection-state.md` 把 source item seen dedupe 和
rejection cooldown 迁到了 Go `agent-runtime`。这些记录在读路径会按 TTL 判断，
但如果 Go JSON store 长期运行不清理，过期记录会持续增长。

retention cleanup 是确定性运行时维护逻辑，应该由持有该状态的 Go 侧负责。

## 范围

Go 负责清理：

- proactive delivery records。
- source item seen records。
- context-only timestamp records。
- rejection cooldown records。

Go 不负责清理：

- Python SQLite semantic items。
- proactive tick log / tick step log。
- session last markers，例如 `drift_last_at` 和 `context_only_last_at`。
- AnyAction quota 历史窗口，后续如有多窗口存储再单独设计。

## API

```text
POST /v1/proactive/cleanup
```

请求：

```json
{
  "seen_ttl_hours": 24,
  "delivery_ttl_hours": 48,
  "context_only_ttl_hours": 24,
  "rejection_cooldown_ttl_hours": 2,
  "timestamp": "2026-05-31T12:00:00Z"
}
```

响应：

```json
{
  "removed_deliveries": 1,
  "removed_seen_items": 1,
  "removed_context_only": 1,
  "removed_rejection_cooldowns": 1,
  "timestamp": "2026-05-31T12:00:00Z",
  "side_effect": "runtime_state_cleanup"
}
```

## Python Bridge

`AgentRuntimeProactiveStateStore.cleanup(...)`：

1. 先调用 Go `/v1/proactive/cleanup` 清理 Go-owned JSON state。
2. 无论 Go 成功与否，继续执行 SQLite fallback cleanup，保持历史本地库清理。
3. Go 请求失败只记录 warning，不阻断 proactive loop。

## 领域规则

- TTL 小于等于 0 时使用安全默认值：delivery/seen/context-only 默认 24 小时。
- rejection cooldown TTL 小于等于 0 表示不清理 rejection cooldown。
- 清理只删除 strictly before cutoff 的记录，避免边界时间重复抖动。

## 验收

- Go service/store/http targeted tests 覆盖清理计数和 fresh record 保留。
- Python bridge test 覆盖 `/v1/proactive/cleanup` 请求体和 fallback 继续执行。
- 全量 `go test ./...`、`go vet ./...`、构建和 Python targeted test 通过。
