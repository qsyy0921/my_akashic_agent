# ATDD: Agent Job External Lease Launcher Bundle

## User-visible contract

- 维护者可以先读 `GET /v1/agent-job-external-lease/launcher-bundle`，拿到 canonical
  repo-local launcher contract，而不是手工拼 external-lease flags。
- 这条 bundle 本身不改 runtime，只提供 script path、flags、required external inputs
  和 verification steps。
- repo-owned live smoke 会直接消费这条 bundle，再验证 temp runtime promoted state。

## Acceptance checks

1. blocked runtime 上 bundle endpoint 可读，且 `reason` 为
   `agent_job_external_lease_launcher_bundle_blocked`。
2. bundle 暴露 `script_path=.\\scripts\\start-agent-runtime.ps1`。
3. bundle 暴露 canonical launcher flags。
4. bundle 显式要求 `QueueDSN` 作为 required secret external input。
5. 用 bundle 启动的 temp runtime 会在 `queue_backend / queue_topology`
   上显示 result-ack promoted ownership。
6. worker coverage 满足后，bundle 会变为 ready。
7. approval-bound preflight 仍会从 `missing_approval_id` 到
   `agent_job_external_lease_preflight_ready`。
