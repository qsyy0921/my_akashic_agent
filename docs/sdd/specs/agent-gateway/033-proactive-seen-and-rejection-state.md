# 033 Proactive Seen And Rejection State

Date: 2026-05-31

## 背景

Python proactive loop 里仍有两类确定性状态留在 SQLite：

- source item seen dedupe：避免同一来源条目在 TTL 内反复进入候选。
- rejection cooldown：模型或策略拒绝过的条目，在冷却窗口内不再重复尝试。

这两类状态不涉及 LLM 推理和 prompt 选择，适合交给 Go `agent-runtime` 统一持久化、去重和跨进程共享。

## 范围

迁移到 Go 优先路径：

- `is_item_seen`
- `mark_items_seen`
- `is_rejection_cooled`
- `mark_rejection_cooldown`

继续留在 Python/SQLite：

- semantic items 的文本候选缓存。
- tick log / tick step log。
- `bg_context_last_main_at`。
- cleanup 的 SQLite 本地清理。

## 领域规则

- source item key 由 `(source_key, item_id)` 组成。
- `source_key` 兼容历史 Python 规则：`mcp:*:*` 归一到前两段，例如 `mcp:news:feed-a` 和 `mcp:news:feed-b` 都按 `mcp:news` 去重。
- seen 判断使用 `seen_at >= now - ttl_hours`。
- rejection cooldown 判断使用 `rejected_at >= now - ttl_hours`。
- rejection `ttl_hours <= 0` 固定返回 not cooled，且 mark 时不写入。

## API

```text
POST /v1/proactive/seen-items
GET  /v1/proactive/seen-items/seen?source_key=...&item_id=...&ttl_hours=24

POST /v1/proactive/rejection-cooldowns
GET  /v1/proactive/rejection-cooldowns/cooled?source_key=...&item_id=...&ttl_hours=24
```

写入请求格式：

```json
{
  "entries": [
    {"source_key": "mcp:news:feed-a", "item_id": "item-a"}
  ],
  "timestamp": "2026-05-30T10:00:00Z"
}
```

rejection cooldown 额外包含：

```json
{"hours": 2}
```

## Python Bridge

`AgentRuntimeProactiveStateStore` 继续作为组合适配器：

1. 读操作先问 Go；Go 命中则直接返回 true。
2. Go 未命中时继续查 SQLite fallback，避免迁移期间丢历史状态。
3. 写操作先写 Go，再写 SQLite，让 fallback 保持 warm。
4. Go 不可用、HTTP 错误或响应结构异常时，记录 warning 并回退 SQLite。

## 验收

- Go domain/app/store/http targeted tests 覆盖 TTL、MCP source 归一、持久化和 endpoint。
- Python bridge mock runtime 覆盖 seen/rejection 的 Go route、参数、写入 fallback。
- Runtime 500 时 SQLite fallback 仍生效。
