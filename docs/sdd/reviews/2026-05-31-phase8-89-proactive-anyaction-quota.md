# Phase 8.89 Review: Go-owned Proactive AnyAction Quota

日期：2026-05-31

## 结论

通过。AnyAction quota 是确定性 admission 状态，适合由 Go `agent-runtime` 持久化和暴露诊断；Python 仍负责主动推送策略、概率计算和 LLM 行为，边界清楚。

## 设计审查

- 复用现有 `ProactiveStateService`，没有为一个小状态新增服务进程。
- quota rollover 规则在 Go app/domain 层执行，存储层只负责 JSON 持久化。
- `GET /v1/proactive/anyaction/quota` 可能执行 rollover 写入，因此 `side_effect` 明确为 `runtime_state_rollover` 或 `runtime_state_read_rollover_if_needed`。
- Python `AgentRuntimeAnyActionQuotaStore` 保持 fallback：Go 不可用时继续使用原 `proactive_quota.json`，不会阻断主动推送循环。

## 风险

- 当前只迁移 quota 状态，不迁移 probability gate；如果后续需要跨进程一致的随机 draw，需要单独设计 Go-owned admission decision。
- Python fallback 和 Go runtime 都记录 action 时，短期会出现双写；读取时取同窗口更保守的 `used=max(runtime, fallback)`，避免放宽 quota。

## 验收门

- `go test ./app/service ./infrastructure/proactivestate ./trigger/http ./infrastructure/memory`
- `uv run pytest tests/test_agent_runtime_proactive_state.py`
- 后续 live 验证 proactive 触发后 Go `proactive-state.json` 中 `anyaction_quotas.used` 增长。
