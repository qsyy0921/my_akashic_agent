# 036 Scheduler Job Store

Date: 2026-05-31

## 背景

`SchedulerService` 的 tick loop、软实时提前触发、AI 执行和 message push 仍依赖 Python event loop 与模型调用。但 `schedules.json` 是确定性的运行时状态：新增、取消、执行后重排都会把完整 job snapshot 写盘。

这类 durable snapshot 适合迁移给 Go `agent-runtime` 统一持久化和观测，避免重启后不同 Python 进程各自持有一份调度状态。

## 范围

迁移到 Go：

- scheduled job snapshot 的校验、完整替换、列表查询。
- 默认文件态 `.akashic-workspace/agent-runtime/scheduler-jobs.json`。
- HTTP contract：Python `JobStore` 保存时同步 snapshot 到 Go，读取时优先读 Go。

继续留在 Python：

- `SchedulerService` tick loop。
- `compute_fire_at` / cron / latency tracker。
- instant/soft job 执行。
- AI prompt、LLM 调用和最终平台发送。

## API

```text
GET  /v1/scheduler/jobs
POST /v1/scheduler/jobs/snapshot
```

Snapshot 请求：

```json
{
  "source": "python_scheduler",
  "jobs": [
    {
      "id": "job-1",
      "trigger": "after",
      "tier": "instant",
      "fire_at": "2026-06-01T09:00:00Z",
      "channel": "qq",
      "chat_id": "1049511700",
      "message": "提醒",
      "timezone": "Asia/Shanghai",
      "created_at": "2026-06-01T08:00:00Z",
      "enabled": true
    }
  ]
}
```

Go 返回：

```json
{
  "count": 1,
  "source": "python_scheduler",
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
  -> domain/model.SchedulerJob
```

## Python Bridge

`JobStore` 增加 `runtime_config`：

1. `load()`：runtime enabled 时先读 `GET /v1/scheduler/jobs`；Go 返回非空时使用 Go；Go 为空且本地 JSON 非空时保留本地迁移 fallback。
2. `save()`：先向 `POST /v1/scheduler/jobs/snapshot` 写完整 snapshot，再写本地 `schedules.json` fallback。
3. Go 不可用时记录 warning 并继续使用本地 JSON，避免调度功能被 runtime 短暂不可用打断。

## 验收

- Go domain/app/store/http tests 覆盖 snapshot replace、validation、持久化和 HTTP contract。
- Python `JobStore` mock runtime 覆盖 load、save 和失败 fallback。
- 全量 Go test 与 scheduler Python targeted tests 通过。
