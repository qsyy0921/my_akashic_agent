# Phase 8.90 Review: Go-owned Proactive Seen And Rejection State

日期：2026-05-31

## 结论

通过。source item seen dedupe 和 rejection cooldown 都是确定性 runtime 状态，适合由 Go `agent-runtime` 持久化并暴露统一 API；Python 仍负责主动推送策略、候选语义缓存、tick log 和模型推理。

## 设计审查

- 复用现有 `ProactiveStateService` 和 `proactivestate.Store`，没有新增服务进程。
- 领域层保留历史 `mcp:*:* -> mcp:*` 归一规则，避免迁移后同源 MCP 条目重复进入候选。
- Python bridge 采用 Go-first + SQLite warm fallback：Go 命中更保守，Go 未命中仍读历史 SQLite。
- rejection `ttl_hours <= 0` 明确为禁用冷却，不写入、不命中，保持 Python 旧行为。

## 风险

- cleanup 在后续 `Phase 8.91` 已迁入 Go-owned retention cleanup；本切片实现时的过期记录增长风险已收敛。
- semantic items 仍留 Python/SQLite，因为它是 AI 候选缓存，后续如要迁移需要和 RAG/memory 评估一起设计。

## 验收门

- `go test ./domain/model ./app/service ./infrastructure/proactivestate ./infrastructure/memory ./trigger/http`
- `uv run pytest tests/test_agent_runtime_proactive_state.py`
- 后续 live 验证 proactive loop 触发后 `.akashic-workspace/agent-runtime/proactive-state.json` 中出现 `seen_items` 或 `rejection_cooldowns`，且 SQLite fallback 不放宽去重。
