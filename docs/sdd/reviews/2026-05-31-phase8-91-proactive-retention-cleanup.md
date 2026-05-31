# Phase 8.91 Review: Go-owned Proactive Retention Cleanup

日期：2026-05-31

## 结论

通过。Go `agent-runtime` 现在不仅持有 proactive deterministic state，也负责该状态的 TTL retention cleanup；Python 继续保留 SQLite fallback cleanup，边界清晰。

## 设计审查

- 清理逻辑在 `ProactiveStateService` 计算 cutoff，存储层只按 cutoff 删除并返回计数。
- JSON store 和 memory store 实现同一 `CleanupProactiveState` 出站端口，没有绕过 app/domain 规则。
- session last markers 和 AnyAction quota 不被 cleanup 删除，避免破坏主动推送节流与 quota 判断。
- Python bridge 先清 Go、再清 SQLite；Go 不可用时不阻断 proactive loop。

## 风险

- 当前 cleanup 由 Python proactive loop 触发，不是 Go 后台定时任务。这样能先复用已有生命周期，避免新增 runner；后续如果 proactive loop 不常运行，再考虑 Go-owned periodic cleanup worker。
- semantic items 仍由 SQLite cleanup 管理，这是 AI 候选缓存，暂不迁入 Go。

## 验收门

- `go test ./domain/model ./app/service ./infrastructure/proactivestate ./infrastructure/memory ./trigger/http`
- `uv run pytest tests/test_agent_runtime_proactive_state.py`
- 后续 live 验证 proactive loop cleanup 后 Go `proactive-state.json` 中过期 `seen_items` / `rejection_cooldowns` 会减少。
