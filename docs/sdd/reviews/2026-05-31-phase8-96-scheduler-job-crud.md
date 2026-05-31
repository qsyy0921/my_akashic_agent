# Phase 8.96 Review: Go-owned Scheduler Job CRUD

日期：2026-05-31

## 结论

通过。该切片把 tool-driven schedule 新增和取消从 Python 全量 snapshot 替换收敛到 Go 单任务 upsert/delete，降低多 Python 进程或多轮工具调用之间互相覆盖 scheduler job 的风险。

## 设计审查

- Go 复用既有 `SchedulerJob` domain validation，新增 mutation 仍走 `trigger/http -> app/service -> port/out -> infrastructure` 分层。
- 新增 `POST /v1/scheduler/jobs/upsert` 与 `DELETE /v1/scheduler/jobs/{job_id}` 只写 scheduler state，不触发 tick、AI、outbox 或平台发送。
- `schedulerjobstore` 和 memory store 都实现单任务 upsert/delete，并保留 snapshot replace 兼容已有执行完成重排路径。
- Python `SchedulerService.add_job` / `cancel_job` / `cancel_job_by_name` 优先调用 `JobStore.upsert/delete`，本地 `schedules.json` 继续作为 fallback 镜像。
- 执行完成后的 recurring reschedule 和 one-shot 删除仍走 snapshot replace，本切片不把 cron/interval 语义或执行结果回写搬进 Go。

## 风险

- 执行完成后的 snapshot replace 仍可能覆盖并发新增/取消，这是下一步应继续拆出的 Go-owned scheduler completion/reschedule mutation。
- Python runtime CRUD 失败时仍写本地 fallback，因此 Go 不可用期间多进程一致性无法保证；这与现有 fallback 策略一致。
- HTTP DELETE 当前按 job id 精确删除，不支持按 name 删除；按 name 的匹配仍在 Python 本地 job view 上完成。

## 验收门

- `go test ./app/service ./infrastructure/schedulerjobstore ./trigger/http`
- `go test ./...`
- `go vet ./...`
- `go build -o ..\..\.tmp\bin\agent-runtime.exe .\cmd\agent-runtime`
- `uv run pytest tests\test_job_store.py tests\test_scheduler_service.py -q --basetemp .tmp\pytest-scheduler-crud`
- `uv run python -m py_compile agent\scheduler.py`
- 本地 HTTP smoke：临时 `agent-runtime` 使用 `.tmp\scheduler-crud-smoke-runtime`，验证 upsert create、upsert update、delete found、delete missing、list 和 diagnostics；不触发 QQ/Telegram 发送。
