# Phase 8.93 Review: Go-owned Scheduler Job Store

日期：2026-05-31

## 结论

通过。当前切片只迁移调度任务的 durable snapshot，不迁移 Python scheduler tick loop 或 AI 执行，因此不会改变现有提醒、软实时任务和 message push 行为。

## 设计审查

- Go `SchedulerJob` 是显式领域模型，校验 trigger/tier/channel/chat_id/fire_at，以及 instant/soft 所需 message/prompt。
- HTTP 层只提供完整 snapshot replace 和 list，避免引入半成品的跨进程 scheduler 控制命令。
- Python `JobStore` 仍写本地 `schedules.json` fallback，Go 为空时会读取本地旧数据，迁移期间不会丢历史调度。
- Go 默认文件态使用 `scheduler-jobs.json`，与其它 runtime state 一致，可通过 `AKASHIC_SCHEDULER_JOBS_DSN/PATH` 覆盖。
- 该切片没有触发真实 QQ/Telegram 发送，也不启动新的服务进程。

## 风险

- Python scheduler 仍是单进程执行者；如果未来要多 Python worker 竞争执行 scheduled jobs，需要继续设计 Go-owned lease/fencing，而不是直接让多个 tick loop 读同一 snapshot。
- 当前 Go snapshot 是完整替换模型；后续若开放外部 CRUD，需要补 per-job version 或 compare-and-swap，避免覆盖 Python 正在执行后的 reschedule 结果。
- soft job 的 LLM 执行仍在 Python，耗时/失败恢复仍沿用原 SchedulerService 行为。

## 验收门

- `go test ./...`
- `uv run pytest tests/test_job_store.py tests/test_scheduler_service.py tests/test_schedule_tool.py`
- `go vet ./...`
- `go build -o .tmp/bin/agent-runtime.exe ./cmd/agent-runtime`
- 本地 runtime smoke：`POST /v1/scheduler/jobs/snapshot` 后 `GET /v1/scheduler/jobs` 可读，文件态生成 `scheduler-jobs.json`。
