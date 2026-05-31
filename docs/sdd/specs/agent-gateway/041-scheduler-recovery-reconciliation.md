# 041 Scheduler Recovery Reconciliation

Date: 2026-05-31

## 背景

`036` 到 `040` 已经把 scheduler job snapshot、diagnostics、execution lease、
tool-driven CRUD 和 completion mutation 迁移到 Go-owned runtime state。Python
仍负责 tick loop、misfire 恢复、cron/interval next fire 计算和实际 AI/推送执行。

当前 `SchedulerService.load_and_recover()` 在启动时会：

- 将 missed recurring job 的 `fire_at` 推进到未来时间。
- 丢弃超过 grace window 的 one-shot job。

但这些变更只发生在 Python 内存里，没有回写 Go `scheduler-jobs.json` 或本地
`schedules.json`。因此 Go diagnostics 可能继续显示旧 overdue job，重启后也可能
反复恢复同一批 stale job。

## 范围

接入到 Python：

- `load_and_recover()` 先构造最终 recovered job map。
- 对启动恢复时推进过的 recurring job，调用现有 `JobStore.upsert()`。
- 对启动恢复时丢弃的 expired one-shot job，调用现有 `JobStore.delete()`。
- 每次 mutation 都传入最终 recovered job map，让本地 mirror 与 Go state 一致。

复用现有 Go：

- `POST /v1/scheduler/jobs/upsert`
- `DELETE /v1/scheduler/jobs/{job_id}`

暂不迁移：

- scheduler tick loop。
- due job selection。
- AI execution / message push。
- cron/interval next fire 计算。
- execution lease acquisition/completion semantics。

## 规则

- startup recovery 不触发 QQ/Telegram 平台发送。
- startup recovery 不申请 execution lease，因为它不是 due job execution。
- recurring misfire 只做单 job upsert，不做全量 snapshot replace。
- expired one-shot 只做单 job delete，不做全量 snapshot replace。
- mutation 必须在完整 recovered map 构造后执行，避免本地 mirror 写入半成品。
- Go runtime 不可用时，`JobStore.upsert/delete` 继续写本地 fallback。

## 分层

```text
Python SchedulerService.load_and_recover
  -> Python JobStore.upsert/delete
  -> Go trigger/http scheduler job CRUD
  -> Go app/service SchedulerJobService
  -> Go infrastructure schedulerjobstore
```

## 验收

- Local-only tests 覆盖 recurring misfire 推进后写回 `schedules.json`。
- Local-only tests 覆盖 expired one-shot 丢弃后从 `schedules.json` 删除。
- Runtime-enabled tests 覆盖 `load_and_recover()` 从 Go 读取 jobs 后调用
  `/v1/scheduler/jobs/upsert` 和 `DELETE /v1/scheduler/jobs/{job_id}`，且不调用
  `/v1/scheduler/jobs/snapshot`。
- Python py_compile 通过。
- Go test/vet/build 保持通过。
