# 014 Python Proactive State Runtime Bridge

Date: 2026-05-30

## 背景

`013-proactive-scheduling-state.md` 已经在 Go `agent-runtime` 中实现了
proactive scheduling state。剩余问题是 Python proactive loop 仍然只读写
SQLite，因此 Go 侧 API 还没有成为运行时路径。

本切片把 Python 的确定性调度状态接到 Go，同时保持 SQLite fallback。

## 范围

迁移到 Go 优先路径：

- `is_delivery_duplicate`
- `mark_delivery`
- `count_deliveries_in_window`
- `is_item_seen`
- `mark_items_seen`
- `is_rejection_cooled`
- `mark_rejection_cooldown`
- `get_last_context_only_at`
- `mark_context_only_send`
- `count_context_only_in_window`
- `get_last_drift_at`
- `mark_drift_run`

继续留在 SQLite：

- tick log / tick step log
- semantic items
- `bg_context_last_main_at`
- cleanup

## 设计

```text
bootstrap/proactive.py
  -> _build_proactive_state_store(config, workspace)
      -> ProactiveStateStore(workspace/proactive.db)
      -> AgentRuntimeProactiveStateStore(config.agent_runtime, fallback)
```

`AgentRuntimeProactiveStateStore` 是组合适配器：

1. 对确定性 scheduling methods 优先调用 Go `/v1/proactive/*`。
2. 写操作在调用 Go 后继续写 SQLite，保持 fallback warm。
3. Go 不可用、响应错误或返回结构异常时，读写回退到 SQLite。
4. Go 可用但历史数据尚未导入时，读操作会用 SQLite 做迁移桥接：
   delivery duplicate 可由 SQLite 命中，窗口计数取 Go/SQLite 的较大值，last
   timestamp 取更新的一侧。
5. 非迁移方法直接委托 SQLite，避免影响 dashboard tick log 和 AI 候选逻辑。

## 启用条件

只有同时满足以下条件才启用 runtime-backed state：

- `config.agent_runtime.enabled == true`
- `config.agent_runtime.base_url` 非空

否则保持历史 `ProactiveStateStore`。

## 失败策略

Go runtime 是优化路径，不是主动推送链路的单点依赖：

- 查询失败：用 SQLite 当前状态判断。
- 写入失败：记录 warning，并写 SQLite。
- SQLite 失败：按历史行为抛出异常。

## 验收

- Mock runtime 覆盖所有迁移方法的 route、参数和返回值。
- Runtime 500 时确认 SQLite fallback 生效。
- bootstrap helper 在 `agent_runtime.enabled=true` 时返回 runtime-backed store。
- Python targeted tests 通过。
