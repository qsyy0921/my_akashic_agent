# Phase 8.94 Review: Go-owned Scheduler Diagnostics

日期：2026-05-31

## 结论

通过。该切片只增加 Go 对 scheduler snapshot 的只读诊断和 runtime overview 聚合，不改变 Python scheduler 的执行语义，也不触发平台发送。

## 设计审查

- 复用 `SchedulerJobService`，没有新增服务进程或新的部署边界。
- 新增 `SchedulerJobDiagnosticsViewer` 入站端口，HTTP 层只读查询，不开放外部执行/取消控制。
- 诊断口明确 `side_effect=none`，并按 overdue/due_soon/disabled/future 聚合运行状态。
- Runtime overview 只消费 Go-owned snapshot store，不直接读取 Python `schedules.json`，避免 dashboard 重新耦合 Python 本地文件。
- Python runtime overview dashboard 主路径透传 Go aggregate；旧 fallback 只读调用 `/v1/scheduler/diagnostics`，不会触发 scheduler tick 或平台发送。

## 风险

- 多 Python scheduler tick loop 竞争执行的问题仍未解决；本切片只是观测能力，后续如果要多进程执行，需要继续设计 Go-owned tick lease/fencing。
- overdue 只代表 Go snapshot 中 `fire_at <= now`，不等价于确认 Python 执行失败；如果 Python worker 停止，runtime worker/status 诊断也应一起看。

## 验收门

- `go test ./app/service ./trigger/http ./cmd/agent-runtime`
- `go test ./...`
- `go vet ./...`
- `go build -o ..\..\.tmp\bin\agent-runtime.exe .\cmd\agent-runtime`
- `uv run pytest tests\test_runtime_overview_dashboard_plugin.py -q --basetemp .tmp\pytest-runtime-overview-scheduler`
- 本地 HTTP smoke：临时 `agent-runtime` 写入 2 条 scheduler snapshot，`GET /v1/scheduler/diagnostics` 返回 `overdue_jobs=1`、`due_soon_jobs=1`，`GET /v1/runtime-overview` 返回 `scheduler_jobs=2` 且 `Scheduler Jobs` card 为 `warn`；state file 写入 `.tmp\scheduler-diagnostics-smoke-runtime\scheduler-jobs.json`。
