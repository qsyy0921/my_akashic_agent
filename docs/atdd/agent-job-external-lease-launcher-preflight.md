# ATDD: Agent Job External Lease Launcher Preflight

## User-visible contract

- 维护者可以只通过 repo-local `scripts/start-agent-runtime.ps1` 起一份隔离 runtime，
  不需要手工 export 一串 external-lease flags。
- 起起来的 temp runtime 会在 `/v1/runtime-config`、`/v1/queue-backend`、
  `/v1/queue-topology` 和 `/v1/agent-job-external-lease/preflight`
  上体现 result-ack preflight 所需边界。

## Acceptance checks

1. launcher 参数可显式注入 queue/result-ack flags。
2. `/v1/runtime-config` 可见这些 flags，且布尔项不是 secret。
3. `/v1/queue-backend` 显示 `provider=nats_jetstream`。
4. `/v1/queue-topology` 的 `agent_job.execution_owner` 为
   `python_ai_worker_with_nats_result_ack`，`ack_owner` 为
   `nats_external_lease_result_ack`。
5. 不带 approval 的 preflight 返回 `missing_approval_id`。
6. 注入 active approval 后 preflight 返回
   `agent_job_external_lease_preflight_ready`。
