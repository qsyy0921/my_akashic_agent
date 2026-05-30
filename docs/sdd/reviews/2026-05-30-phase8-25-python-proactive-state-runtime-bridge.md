# Phase 8.25 Python Proactive State Runtime Bridge Review

Date: 2026-05-30

## 设计结论

Approved. Python proactive loop 现在可以在 `agent_runtime.enabled=true` 时把
delivery 去重、窗口计数、context-only 节流和 drift 间隔标记交给 Go
`agent-runtime`，同时保留 SQLite 作为兼容 fallback。

## 边界

- Go 负责 deterministic scheduling state。
- Python 继续负责 prompt、LLM 决策、semantic candidates、tick log 和 dashboard
  历史视图。
- SQLite fallback 仍然存在，避免 runtime 重启、端口未启动或 API 错误时打断主动链路。

## 实现摘要

- Added `integrations.agent_runtime_proactive_state.AgentRuntimeProactiveStateStore`。
- `bootstrap.proactive._build_proactive_state_store` 在 runtime 启用时返回组合 store。
- Runtime-backed store 对写操作执行 Go + SQLite 双写；读操作优先 Go，失败时回退 SQLite。
- Go 可用但缺少历史数据时，用 SQLite 做迁移桥接，避免启用瞬间丢失旧的冷却窗口。
- 非迁移状态方法直接委托原 SQLite `ProactiveStateStore`。

## 风险

1. 这是同步 HTTP 调用，运行在 proactive loop 中；目前请求 timeout 走
   `agent_runtime.request_timeout_seconds`，后续可按需加入更短的 state timeout。
2. 历史 SQLite delivery/context/drift 状态不会一次性导入 Go；读侧迁移桥接只覆盖当前窗口判断。
3. dashboard proactive tick log 仍来自 SQLite，这是有意保留的边界。

## 验证

- `uv run pytest tests/test_agent_runtime_proactive_state.py -q`
- `uv run pytest tests/test_more_support_modules.py::test_bootstrap_proactive_builders_cover_enabled_and_disabled_paths tests/test_proactive_facade_phase4.py::test_build_proactive_runtime_accepts_facade_memory -q`
- `uv run pytest tests/proactive_v2 -q`
- `uv run python -m py_compile bootstrap/proactive.py integrations/agent_runtime_proactive_state.py`

## 后续

- 给 runtime-backed proactive state 增加 dashboard health 指示。
- 评估是否把 tick log 也拆出为 Go-owned append-only event stream。
