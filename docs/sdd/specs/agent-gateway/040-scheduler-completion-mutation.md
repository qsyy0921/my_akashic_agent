# 040 Scheduler Completion Mutation

Date: 2026-05-31

## 背景

`039-scheduler-job-crud` 已经把 tool-driven schedule add/cancel 改为 Go 单任务
upsert/delete，但 Python scheduler 执行完成后仍会 `JobStore.save(self._jobs)` 全量
替换 snapshot，再单独 release execution lease。

这留下两个确定性状态风险：

- 执行完成后的全量 snapshot 可能覆盖其它进程刚新增或取消的 job。
- job state update 和 lease release 是两个 HTTP 请求；如果执行者 lease 已过期或
  token 不匹配，旧执行者仍可能写入 snapshot。

完成态回写、fencing 和 lease release 是 Go 适合承接的基础设施逻辑。Python 仍负责
实际执行、run_count 增加和 recurring next fire 计算。

## 范围

新增到 Go：

- `POST /v1/scheduler/jobs/{job_id}/complete`
- 按 `holder_id + lease_token` 校验 active execution lease。
- 对 recurring job 执行单任务 reschedule/upsert。
- 对 one-shot job 执行单任务 delete。
- 成功写入 job state 后释放 execution lease。

接入到 Python：

- `_execute_and_reschedule` 在拿到 Go lease token 时调用 complete mutation。
- Go complete 成功后只镜像写入本地 `schedules.json`，不再全量替换 Go snapshot。
- complete 失败时保留 lease 到 TTL，避免错误 release 后被其它进程立刻重复执行。

暂不迁移：

- scheduler tick loop。
- due job selection。
- cron/interval next fire 计算。
- soft job LLM 调用和 instant message push。

## API

```text
POST /v1/scheduler/jobs/{job_id}/complete
```

Recurring reschedule request:

```json
{
  "source": "python_scheduler",
  "holder_id": "scheduler:akashic-python-worker:abc123",
  "lease_token": "opaque-token",
  "action": "reschedule",
  "job": {
    "id": "schedule:abc",
    "trigger": "every",
    "tier": "soft",
    "fire_at": "2026-05-31T12:30:00Z",
    "channel": "telegram",
    "chat_id": "100",
    "prompt": "天气",
    "run_count": 4,
    "enabled": true
  }
}
```

One-shot delete request:

```json
{
  "source": "python_scheduler",
  "holder_id": "scheduler:akashic-python-worker:abc123",
  "lease_token": "opaque-token",
  "action": "delete"
}
```

Response:

```json
{
  "job_id": "schedule:abc",
  "action": "reschedule",
  "deleted": false,
  "lease_released": true,
  "side_effect": "runtime_state_write"
}
```

## 分层

```text
trigger/http
  -> app/port/in/SchedulerJobManager
  -> app/service/SchedulerJobService
  -> app/port/out/SchedulerJobRepository
  -> app/port/out/SchedulerExecutionLeaseRepository
  -> infrastructure/schedulerjobstore + infrastructure/schedulerleasestore
```

## 规则

- complete 必须携带 `holder_id` 和 `lease_token`。
- lease 必须存在、未过期、且 holder/token 匹配。
- `action=reschedule` 必须携带完整 job，且 body job id 必须等于 path job id。
- `action=delete` 删除 job，缺失 job 也视为成功删除完成态。
- Go complete 成功后删除 lease；响应不返回 raw lease token。
- complete 失败时 Python 不调用 release，让 lease 到 TTL 后自然恢复。

## 验收

- Go service/http tests 覆盖 reschedule 成功并释放 lease、delete 成功并释放 lease、
  token mismatch 拒绝且不改 job。
- Python scheduler tests 覆盖执行完成后调用 complete mutation；complete 失败时不
  release lease 且本地 fallback 镜像仍更新。
- HTTP smoke 只写 synthetic scheduler job/lease state，不触发 QQ/Telegram 发送。
