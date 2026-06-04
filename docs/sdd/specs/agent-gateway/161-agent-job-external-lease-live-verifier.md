# 161 Agent Job External Lease Live Verifier

## Context

- `agent_job` external lease result-ack 已有 Go readiness 和 plan endpoint。
- 但当前 end-of-turn goal 更新仍需要人工拼接多个 endpoint，才能解释：
  - 是缺 provider / DSN / queue mode
  - 还是缺 strict token / result-ack flag
  - 还是缺 duplicate/flow smoke
  - 还是 worker coverage 不够
- 这类残留项本质上是 Go control-plane cutover 问题，应该有 repo-owned 的
  live verifier。

## Decision

新增：

```text
scripts/verify-agent-job-external-lease.ps1
```

脚本必须一次性聚合：

- `/v1/agent-job-external-lease/readiness`
- `/v1/agent-job-external-lease/plan`
- `/v1/runtime-config`
- `/v1/queue-backend`
- `/v1/queue-topology`
- `/v1/runtime-overview`
- `/v1/agent-worker-statuses`

并把当前 blocker 拆成至少四类：

1. `configuration`
2. `execution_owner`
3. `smoke`
4. `worker_coverage`

## Requirements

1. verifier 必须输出当前 execution scope 是：
   - `state_store_lease`
   - 或 `nats_result_ack`
2. verifier 必须显式输出以下 flags 是否存在：
   - `AKASHIC_QUEUE_BACKEND`
   - `AKASHIC_QUEUE_DSN`
   - `AKASHIC_QUEUE_MODE`
   - `AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER`
   - `AKASHIC_QUEUE_DUAL_READ_SMOKE_PASSED`
   - `AKASHIC_QUEUE_STATE_LEASE_WORKERS_DISABLED`
   - `AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN`
   - `AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED`
   - `AKASHIC_QUEUE_AGENT_JOB_DUPLICATE_SMOKE_PASSED`
   - `AKASHIC_QUEUE_AGENT_JOB_FLOW_SMOKE_PASSED`
3. verifier 必须明确区分：
   - 当前 provider 是否已是 `nats_jetstream`
   - 当前 selected provider 是否支持 `agent_job result-ack`
   - 当前 queue topology 的 `agent_job` owner 是否仍是
     `python_ai_worker_state_store_lease`
4. unified goal verifier 必须复用这个脚本，而不是只保留简写摘要。

## Non-Goals

- 不启用 NATS external lease。
- 不更改 queue provider。
- 不 ack/nack/term MQ。
- 不执行 Python AI job。
