# Phase 8.101 Review: Dashboard Proactive Tick Log Runtime Fallback

日期：2026-05-31

## 结论

通过。该切片补齐 Go `/v1/proactive/tick-logs` 的 dashboard 查询能力，并让
Python dashboard proactive tick log 页面在 SQLite mirror 无匹配记录时回退读取 Go runtime。
这保持了旧 dashboard 兼容路径，也让 Go-owned tick audit state 在 mirror 缺失时可见。

## 设计审查

- Go 只增强确定性只读查询：offset、started_at 区间、sort_by、sort_order，不参与 proactive 推理或发送。
- Python dashboard 采用 SQLite-first / Go-fallback：已有 mirror 不受影响，runtime 不可用时不阻断页面。
- fallback 覆盖 list、detail、steps；overview 仍读 SQLite，避免本切片把统计口径一次性切到 Go。
- 查询能力仍在现有 `ProactiveStateService` / repository 端口内完成，没有新增服务或绕过 DDD 分层。

## 风险

- SQLite-first 期间，如果 SQLite 有旧记录但 Go 有更新记录，dashboard list 仍优先展示 SQLite；这是兼容策略，后续确认 live 稳定后再评估 Go-first。
- Go store 当前 retention 为最近 500 tick，dashboard fallback 不应用于长期审计归档。
- runtime fallback 使用短超时，避免 agent-runtime 未启动时拖慢 dashboard，但网络抖动时可能直接回到 SQLite。

## 验收门

- `go test ./app/service ./infrastructure/proactivestate ./infrastructure/memory ./trigger/http`
- `uv run pytest tests/test_dashboard_api.py -q --basetemp .tmp/pytest-dashboard-proactive-runtime-fallback`
- 后续 live 验证：隔离 SQLite mirror 后 dashboard tick list/detail/steps 能从 Go runtime 返回。
