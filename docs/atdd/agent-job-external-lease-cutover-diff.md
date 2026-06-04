# ATDD: agent-job external lease cutover diff

## Acceptance

1. `GET /v1/agent-job-external-lease/cutover-diff` 返回当前 runtime 与
   canonical launcher bundle 的 read-only diff。
2. 当前 8780 runtime 上，diff 明确显示 external-lease/result-ack flags、
   `QueueDSN` 和 `agent_job` owner drift 仍未满足。
3. repo-owned live verifier 能在 temp NATS/runtime 中证明：
   - config drift 先归零；
   - aged `group_memory_extract` pressure 下仍被 worker coverage 阻断；
   - knowledge worker heartbeat 后 diff 变为 ready。
4. unified goal verifier artifact 纳入
   `agent_job_external_lease_cutover_diff` current-turn evidence。
