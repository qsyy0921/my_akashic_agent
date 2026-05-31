# Phase 8.98 Review: Scheduler Recovery Reconciliation

日期：2026-05-31

## 结论

通过。该切片补齐 Python scheduler 启动恢复与 Go-owned scheduler state 之间的
一致性缺口：恢复时推进的 recurring job 和丢弃的 expired one-shot job 都通过
现有 Go CRUD 单任务 mutation 回写，避免 diagnostics 和后续重启继续看到 stale
snapshot。

## 设计审查

- 不新增 Go endpoint，复用已验收的 `/v1/scheduler/jobs/upsert` 和
  `DELETE /v1/scheduler/jobs/{job_id}`。
- Python `load_and_recover()` 先完成 recovered map 构造，再执行 upsert/delete，
  避免本地 fallback mirror 写入部分 job 集合。
- startup recovery 不获取 execution lease，也不调用 completion mutation；它只处理
  启动时的 deterministic schedule state，不执行 due job。
- runtime 不可用时，`JobStore.upsert/delete` 仍写本地 `schedules.json`，保持
  fallback 行为。
- 该切片不迁移 cron/interval 语义到 Go，next fire 仍由 Python 计算，降低行为变化面。

## 风险

- 多个 Python scheduler 同时启动时，startup recovery mutation 仍可能并发；因为每个
  mutation 是单 job CRUD，风险小于全量 snapshot 覆盖，但还不是严格 startup lease。
- Go store 和本地 mirror 仍可能在 runtime 请求失败时短暂分叉；这是兼容 fallback 的
  既有策略。
- disabled job 在当前恢复逻辑中仍不进入内存 job map；本切片不改变该旧语义。

## 验收门

- `uv run pytest tests\test_scheduler_service.py tests\test_job_store.py -q --basetemp .tmp\pytest-scheduler-recovery`
- `uv run python -m py_compile agent\scheduler.py`
- `go test ./...`
- `go vet ./...`
- `go build -o ..\..\.tmp\bin\agent-runtime.exe .\cmd\agent-runtime`
