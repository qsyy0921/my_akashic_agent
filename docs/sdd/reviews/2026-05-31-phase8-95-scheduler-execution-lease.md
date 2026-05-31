# Phase 8.95 Review: Go-owned Scheduler Execution Lease

日期：2026-05-31

## 结论

通过。该切片把 scheduler 执行互斥和 fencing 下沉到 Go，但不迁移 due job 判断、cron 推进、AI 推理或平台发送，因此不会扩大 Go 的行为副作用。

## 设计审查

- 新增 `SchedulerExecutionLease` 领域模型，lease key 固定为 `job_id`，避免同一个 scheduler job 在多个 Python 进程中重复执行。
- Go HTTP 层只提供 acquire / renew / release / list 控制面；写入只影响 runtime lease state，不直接触发任务执行。
- `GET /v1/scheduler/leases` 与 denied acquire 响应不返回 raw token，只暴露 `lease_token_present`，降低 dashboard 或日志泄漏风险。
- Python scheduler 在创建执行 task 前 acquire，执行中按 TTL 续租，保存 Go scheduler snapshot 后 release，避免释放过早导致其它进程读取旧 snapshot 再执行。
- 如果 Go scheduler snapshot 保存失败，Python 会保留 execution lease 直到 TTL 过期，而不是立即 release；这避免 snapshot 短暂失败时发生马上重复执行。
- Go 不负责 scheduler tick loop、cron/interval 语义、LLM 调用、`message_push` 或 proactive 行为，保持 Go/Python 分工清晰。
- Go runtime 不可用时 Python 保留旧本地 `_in_flight` 行为作为 fallback，避免控制面故障导致所有提醒停止。

## 风险

- fallback 模式下仍然只能依赖 Python 进程内 `_in_flight`，多进程重复执行风险会回来；生产环境应把 `agent-runtime` 作为 scheduler lease 的必需控制面。
- TTL 过短可能导致长耗时 soft job 或 snapshot 保存失败后的旧 snapshot 被其它进程重新 acquire；当前 Python 续租周期按 TTL 的三分之一，后续需要用 live 任务观察续租稳定性。
- 本切片没有把 scheduler tick loop 本身迁移到 Go，因此 due-job selection 仍由 Python 本地时钟和 snapshot fallback 共同决定。

## 验收门

- `go test ./...`
- `go vet ./...`
- `go build -o ..\..\.tmp\bin\agent-runtime.exe .\cmd\agent-runtime`
- `uv run pytest tests\test_scheduler_service.py tests\test_job_store.py -q --basetemp .tmp\pytest-scheduler-lease`
- `uv run python -m py_compile agent\scheduler.py`
- 本地 HTTP smoke：临时 `agent-runtime` 使用 `.tmp\scheduler-lease-smoke-runtime`，验证 acquire 成功、第二 holder denied、同 holder renew、release 后 list 不返回 active lease；整个过程不触发 QQ/Telegram 发送。
