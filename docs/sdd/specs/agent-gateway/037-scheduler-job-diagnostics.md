# 037 Scheduler Job Diagnostics

Date: 2026-05-31

## 背景

`036-scheduler-job-store` 让 Go `agent-runtime` 拥有 Python scheduler 的 durable snapshot。下一步需要让这个状态可观测：运维和 dashboard 应能直接从 Go 看到当前调度任务数量、是否有 overdue job、最近即将触发的任务、trigger/tier/channel 分布。

诊断本身是确定性只读基础设施，不涉及 AI 内容决策，也不触发任何平台发送。

## 范围

迁移/新增到 Go：

- scheduler snapshot 的只读聚合诊断。
- runtime overview 中的 scheduler summary 和 card。
- HTTP read-only endpoint。
- Python runtime overview dashboard 对 Go aggregate 的透传字段和旧多接口 fallback 读取。

继续留在 Python：

- scheduler tick loop 和 `_execute_and_reschedule`。
- soft job 的 LLM 执行、latency tracker 和 message push。
- schedule tool 的自然语言解析。

## API

```text
GET /v1/scheduler/diagnostics?limit=50&due_soon_seconds=300
```

响应摘要：

```json
{
  "sampled_jobs": 2,
  "enabled_jobs": 2,
  "overdue_jobs": 1,
  "due_soon_jobs": 1,
  "jobs_by_trigger": {"after": 1, "every": 1},
  "jobs_by_tier": {"instant": 1, "soft": 1},
  "jobs_by_channel": {"qq": 1, "telegram": 1},
  "jobs_by_status": {"overdue": 1, "due_soon": 1},
  "side_effect": "none"
}
```

Runtime overview 新增：

```text
summary.scheduler_jobs
summary.scheduler_jobs_enabled
summary.scheduler_jobs_overdue
summary.scheduler_jobs_due_soon
cards[id=scheduler_jobs]
runtime_overview.scheduler_jobs
```

Python dashboard 主路径仍优先读取 `GET /v1/runtime-overview`。只有 Go aggregate 不可用时，fallback 才单独读取 `GET /v1/scheduler/diagnostics` 并生成同样的 `Scheduler Jobs` card。

## 分层

```text
trigger/http
  -> app/port/in/SchedulerJobDiagnosticsViewer
  -> app/service/SchedulerJobService
  -> app/port/out/SchedulerJobRepository
  -> infrastructure/schedulerjobstore | infrastructure/memory
```

## 状态规则

- `disabled`: job.enabled=false。
- `overdue`: enabled 且 `fire_at <= now`。
- `due_soon`: enabled 且 `now < fire_at <= now + due_soon_window`。
- `future`: 其它 enabled job。

`now` 默认取 Go runtime 当前 UTC 时间，测试和 smoke 可通过 query `timestamp` 固定。

## 验收

- Go service test 覆盖 overdue/due-soon/disabled 聚合。
- HTTP test 覆盖 `GET /v1/scheduler/diagnostics`。
- Runtime overview test 覆盖 summary/card。
- Dashboard test 覆盖 Go aggregate 透传和 fallback 端点兼容。
- endpoint `side_effect=none`，不得触发真实 QQ/Telegram 发送。
