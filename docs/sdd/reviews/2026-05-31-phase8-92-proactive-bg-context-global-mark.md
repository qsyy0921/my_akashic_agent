# Phase 8.92 Review: Go-owned Proactive Background Context Global Mark

日期：2026-05-31

## 结论

通过。`bg_context_last_main_at` 是单纯的全局时间戳状态，适合由 Go `agent-runtime` 统一持久化；Python 仍负责主动推送策略、内容生成和 SQLite fallback。

## 设计审查

- 新增 `ProactiveGlobalMark` 领域模型，限定 key 为 `bg_context_last_main_at`，避免把 global mark 做成任意 KV 垃圾桶。
- 复用现有 `ProactiveStateService` 和 `ProactiveStateRepository`，没有新增服务进程。
- Python bridge 读取时取 Go/SQLite 较新时间，迁移期间不会因为 Go 初始为空而放宽节流。
- session mark、global mark、AnyAction quota 各自建模，避免把不同语义状态混在一个 map 里。

## 风险

- 目前只有一个 global mark。后续如果新增更多全局 proactive marker，需要在 domain 里显式白名单，而不是开放任意 key。
- semantic items 和 tick log 仍在 Python/SQLite，这是有意边界：它们属于 AI 候选缓存和行为审计，不在本切片迁移。

## 验收门

- `go test ./domain/model ./app/service ./infrastructure/proactivestate ./infrastructure/memory ./trigger/http`
- `uv run pytest tests/test_agent_runtime_proactive_state.py`
- 后续 live 验证 proactive background context 触发后 Go `proactive-state.json` 中出现 `global_marks`。
