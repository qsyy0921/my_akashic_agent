# Phase 8.99 Review: Go-owned Proactive Drift State

日期：2026-05-31

## 结论

通过。该切片把 proactive drift 的完成态、recent runs 和 per-skill runtime state
收敛到 Go `ProactiveStateService`，同时保留 Python 本地 JSON mirror。Go 负责
确定性状态和持久化，Python 继续负责 skill 文件扫描、LLM、MCP tool 和 drift 工具执行。

## 设计审查

- 复用现有 proactive state 服务和文件 store，不新增独立服务，符合“不为架构形式过度拆分”的约束。
- 新增 `/v1/proactive/drift/finish` 作为唯一写入口，保证 skill run_count/status/next 与
  recent run 摘要在同一 Go mutation 中更新。
- 新增 `/v1/proactive/drift/summary` 和 `/v1/proactive/drift/skills/{skill_name}` 作为只读诊断/读取入口。
- Python `DriftStateStore` 采用 runtime-first、本地 mirror/fallback 的迁移方式，runtime 不可用时旧行为不变。
- 该切片不迁移 drift skill 文件内容或 tool 文件写入，因为这些属于 workspace 文件系统与 AI 试错边界，不是稳定 runtime 控制面。

## 风险

- Python 与 Go 双写期间可能出现短暂分叉；读取时会合并 runtime 与本地 fallback，降低迁移风险。
- 多 Python 进程同时 finish 同一 skill 时，Go 文件 store 仍是单进程内互斥，不是跨进程数据库事务；当前本地 agent-runtime 单进程部署可接受。
- skill state 的 `status=in_progress` 保持旧语义；后续如果要建更精细的 drift lifecycle，需要单独设计。

## 验收门

- `go test ./app/service ./trigger/http ./infrastructure/proactivestate`
- `go test ./...`
- `go vet ./...`
- `go build -o ..\..\.tmp\bin\agent-runtime.exe .\cmd\agent-runtime`
- `uv run pytest tests\test_agent_runtime_proactive_state.py tests\proactive_v2\test_drift.py -q --basetemp .tmp\pytest-proactive-drift-state`
- `uv run python -m py_compile proactive_v2\drift_state.py integrations\agent_runtime_proactive_state.py proactive_v2\agent_tick_factory.py`
