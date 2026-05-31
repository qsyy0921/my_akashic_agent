# 038 Scheduler Execution Lease

Date: 2026-05-31

## 背景

`036-scheduler-job-store` 已经把 Python scheduler 的 durable snapshot 写入 Go，`037-scheduler-job-diagnostics` 让该状态可观测。但 Python `SchedulerService._tick()` 仍只用进程内 `_in_flight` 防重；如果两个 Python 进程同时运行 scheduler，它们可能同时看到同一个 due job 并各自执行。

这类执行互斥和 fencing 是确定性运行时基础设施，适合下沉到 Go。AI 推理、软实时 prompt 执行、`message_push` 和 cron/interval 语义仍留在 Python。

## 范围

迁移/新增到 Go：

- per scheduler job execution lease 的 acquire / renew / release / list。
- 默认文件态 `scheduler-leases.json`，支持 `AKASHIC_SCHEDULER_LEASES_DSN/PATH` 覆盖。
- HTTP read/write control plane，写入只影响 runtime lease state，不直接执行任务。

接入到 Python：

- Python scheduler 在创建执行 task 前先 acquire 对应 job lease。
- 长任务执行期间按 TTL 周期续租。
- 执行完成、重排或删除 job 并保存 snapshot 后释放 lease。
- 如果 Go scheduler snapshot 保存失败，Python 不主动 release 当前 execution lease，让 lease 至少保留到 TTL，避免其它 scheduler 进程立刻读取旧 snapshot 后重复执行。
- Go 不可用时保留旧行为作为 fallback，避免 scheduler 因控制面故障完全停摆。

继续留在 Python：

- `_tick()` due job 判断、misfire 恢复、cron/interval 推进。
- instant `message_push`。
- soft job 的 agent loop、LLM 调用、latency tracker。

## API

```text
POST /v1/scheduler/leases/acquire
POST /v1/scheduler/leases/renew
POST /v1/scheduler/leases/release
GET  /v1/scheduler/leases
```

Acquire request:

```json
{
  "job_id": "schedule:abc",
  "holder_id": "scheduler:akashic-python-worker:host:pid",
  "ttl_seconds": 300,
  "metadata": {"source": "python_scheduler"}
}
```

Acquire response:

```json
{
  "job_id": "schedule:abc",
  "holder_id": "scheduler:akashic-python-worker:host:pid",
  "lease_token": "opaque-token",
  "lease_token_present": true,
  "active": true,
  "acquired": true,
  "expires_at": "2026-05-31T12:05:00Z",
  "side_effect": "runtime_state_write"
}
```

If another holder has an active lease, Go returns `200 OK` with:

```json
{
  "job_id": "schedule:abc",
  "active": true,
  "acquired": false,
  "denied_reason": "active_lease_held",
  "lease_token_present": true
}
```

List response never returns raw tokens, only `lease_token_present=true/false`.

## 分层

```text
trigger/http
  -> app/port/in/SchedulerJobManager
  -> app/service/SchedulerJobService
  -> app/port/out/SchedulerExecutionLeaseRepository
  -> infrastructure/schedulerleasestore | infrastructure/memory
```

## 规则

- lease key 是 `job_id`，不是 tick loop id；这样长耗时 soft job 也被 fencing。
- 同一 `holder_id` 重复 acquire 视为幂等续租，保留 token 和 acquired_at。
- 不同 holder 遇到未过期 lease，返回 `acquired=false`，Python 必须跳过该 job。
- lease 过期后任意 holder 可重新 acquire。
- renew 必须携带当前 `holder_id` 和 `lease_token`。
- release 必须携带当前 `holder_id` 和 `lease_token`。
- Python 成功保存 Go job snapshot 后再 release，避免其它进程在旧 snapshot 未更新时抢到同一 job。

## 验收

- Go domain/service/http/store tests 覆盖 acquire/deny/renew/release/list。
- Python scheduler tests 覆盖 acquire 后执行、active lease denied 时跳过、执行结束 release。
- Runtime smoke 只写 synthetic scheduler lease state，不触发 QQ/Telegram 发送。
