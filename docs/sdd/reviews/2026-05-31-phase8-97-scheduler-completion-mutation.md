# Phase 8.97 Review: Go-owned Scheduler Completion Mutation

日期：2026-05-31

## 结论

通过。该切片把 scheduler 执行完成后的状态回写改为 Go 侧 lease-fenced mutation，避免 Python 在任务完成后用全量 snapshot 覆盖其它进程的新增/取消，同时把 job state update 与 lease release 收敛到同一个 Go 用例。

## 设计审查

- 新增 `POST /v1/scheduler/jobs/{job_id}/complete`，只接受 `action=reschedule|delete`，不执行 tick、AI 或平台发送。
- Go app service 在写 job state 前校验 active execution lease 的 `holder_id + lease_token`，token mismatch 会拒绝且不改 job。
- `reschedule` 复用 `SchedulerJob` domain validation，并要求 body job id 与 path job id 一致。
- `delete` 删除 one-shot job；job 已不存在也视为完成态删除成功，便于幂等回写。
- 成功写入 job state 后删除 execution lease，响应只暴露 `lease_released=true`，不返回 raw token。
- Python `_execute_and_reschedule` 拿到 Go lease token 时调用 complete mutation；complete 失败只写本地 fallback 镜像，不主动 release lease，让 TTL 保护旧 snapshot。

## 风险

- complete mutation 与 job/lease 两个文件 store 不是严格数据库事务；当前顺序是先写 job state，再删除 lease。若删除 lease 失败，job 已更新但 lease 仍会保留到 TTL。
- Python 本地 fallback 仍可能与 Go 状态短暂不一致；这是 runtime 控制面不可用时的兼容策略。
- next fire 计算仍在 Python，Go 只验证和持久化结果。后续如果要把 cron/interval 语义迁到 Go，需要单独设计，不应混进本切片。

## 验收门

- `go test ./app/service ./trigger/http ./infrastructure/schedulerjobstore`
- `go test ./...`
- `go vet ./...`
- `go build -o ..\..\.tmp\bin\agent-runtime.exe .\cmd\agent-runtime`
- `uv run pytest tests\test_scheduler_service.py tests\test_job_store.py -q --basetemp .tmp\pytest-scheduler-complete`
- `uv run python -m py_compile agent\scheduler.py`
- 本地 HTTP smoke：临时 `agent-runtime` 使用 `.tmp\scheduler-complete-smoke-runtime`，验证 acquire 后 complete reschedule 更新单个 job 并释放 lease、complete delete 删除单个 job 并释放 lease、token mismatch 被拒绝；不触发 QQ/Telegram 发送。
