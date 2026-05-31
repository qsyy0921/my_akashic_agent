# 039 Scheduler Job CRUD

Date: 2026-05-31

## 背景

`036-scheduler-job-store` 已经把 Python scheduler 的 durable snapshot 写入 Go，
`038-scheduler-execution-lease` 解决了 due job 执行互斥。但 Python 在新增或取消
schedule 时仍通过 `JobStore.save(self._jobs)` 全量替换 Go snapshot。

这会留下一个确定性运行时风险：多 Python 进程或多个 tool 调用同时新增/取消任务时，
后写入的 snapshot 可能覆盖先写入的 job。单任务创建、更新、删除是 Go 适合承接的
基础设施状态变更；时间解析、cron 语义、AI 执行和 message push 仍留在 Python。

## 范围

新增到 Go：

- 单个 scheduler job upsert。
- 单个 scheduler job delete。
- 继续复用 `scheduler-jobs.json` 文件态存储和 DDD/六边形分层。

接入到 Python：

- `SchedulerService.add_job` 调用 `JobStore.upsert(job, current_jobs)`。
- `SchedulerService.cancel_job` / `cancel_job_by_name` 调用 `JobStore.delete(job_id, current_jobs)`。
- 本地 `schedules.json` 继续作为 fallback 镜像。

暂不迁移：

- scheduler tick loop。
- due job selection。
- cron/interval 推进。
- soft job LLM 调用和 instant 平台发送。
- 执行完成后的 reschedule/delete snapshot replace。

## API

```text
POST   /v1/scheduler/jobs/upsert
DELETE /v1/scheduler/jobs/{job_id}
```

Upsert request:

```json
{
  "source": "python_scheduler",
  "job": {
    "id": "schedule:abc",
    "trigger": "after",
    "tier": "instant",
    "fire_at": "2026-05-31T12:00:00Z",
    "channel": "telegram",
    "chat_id": "100",
    "message": "提醒",
    "timezone": "Asia/Shanghai",
    "created_at": "2026-05-31T11:00:00Z",
    "enabled": true
  }
}
```

Response:

```json
{
  "job": {"id": "schedule:abc"},
  "source": "python_scheduler",
  "created": true,
  "deleted": false,
  "side_effect": "runtime_state_write"
}
```

Delete response:

```json
{
  "job_id": "schedule:abc",
  "source": "python_scheduler",
  "found": true,
  "deleted": true,
  "side_effect": "runtime_state_write"
}
```

## 分层

```text
trigger/http
  -> app/port/in/SchedulerJobManager
  -> app/service/SchedulerJobService
  -> app/port/out/SchedulerJobRepository
  -> infrastructure/schedulerjobstore | infrastructure/memory
```

## 规则

- Upsert 和 delete 只变更 Go scheduler state，不触发任务执行或平台发送。
- Upsert 复用已有 `SchedulerJob` domain validation。
- Delete 缺失 job 返回 `found=false`，不是错误，方便 Python cancel fallback。
- Python runtime API 失败时仍写本地 `schedules.json`，但不会声称 Go 已同步。
- 执行完成后的 job reschedule 继续使用 snapshot replace，后续再单独设计为
  Go-owned completion/reschedule mutation。

## 验收

- Go store/service/http tests 覆盖 upsert create、upsert update、delete found、
  delete missing，以及 snapshot 兼容。
- Python `JobStore` / `SchedulerService` tests 覆盖 add/cancel 优先调用 Go CRUD、
  Go CRUD 失败时仍保留本地 fallback。
- HTTP smoke 只写 synthetic scheduler job state，不触发 QQ/Telegram 发送。
