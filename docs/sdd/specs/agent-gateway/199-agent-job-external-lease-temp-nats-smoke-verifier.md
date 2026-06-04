# 199 Agent Job External Lease Temp NATS Smoke Verifier

## Context

- `agent_job external lease result-ack` 已有：
  - readiness endpoint
  - cutover plan endpoint
  - live runtime verifier
- 但当前还缺一个 repo-owned、可重复执行的 smoke 入口，直接证明：
  - duplicate terminal notification 会被 `ack`
  - pending/running/succeeded flow 会按 result-ack 语义 `nack/ack`
- 现有 Go smoke test 已经存在，但之前只靠手工 `go test` 命令，且其中
  `pending/running/succeeded` 用了固定历史时间，到了 2026-06-03 会把 lease
  天然跑成 expired，导致 smoke 结论被时间漂移污染。

## Decision

新增 repo-owned verifier：

```text
scripts/verify_agent_job_external_lease_nats_smoke.py
scripts/verify-agent-job-external-lease-nats-smoke.ps1
```

该 verifier 必须：

1. 自动拉起一个临时 `nats:2-alpine -js` 容器；
2. 自动为容器映射随机 localhost 端口；
3. 对 `services/agent-runtime/smoke` 执行：

```text
TestExternalLeaseNATSSmokeAgentJobDuplicateTerminalAck
TestExternalLeaseNATSSmokeAgentJobPendingRunningSucceededFlow
```

4. 输出结构化 JSON，而不是要求人工读 `go test` 文本；
5. 在 unified goal verifier 中留下当前 turn 证据。

同时修复 `services/agent-runtime/smoke/external_lease_nats_smoke_test.go`
里的固定历史时间，改成当前 UTC 时间，避免 smoke 因日期推进而失真。

## Requirements

1. verifier 输出至少包含：
   - `tools`
   - `docker`
   - `go_test`
   - `checks`
   - `conclusion`
2. `checks` 至少包含：
   - `docker_available`
   - `go_available`
   - `temp_nats_ready`
   - `duplicate_terminal_ack_passed`
   - `pending_running_succeeded_flow_passed`
3. `conclusion.category` 至少区分：
   - `repo_owned_temp_nats_smoke_live_verified`
   - `repo_owned_temp_nats_smoke_failed`
   - `docker_not_available`
   - `go_not_available`
   - `temp_nats_not_ready`
4. `scripts/verify-go-migration-goal.ps1` 必须收录该 verifier 输出，并将
   `residual_classification.agent_job_external_lease_result_ack_smoke`
   指向同一结论。
5. verifier 结束后必须清理临时 NATS 容器。

## Non-Goals

- 不修改当前长期运行 runtime 的 queue provider / owner。
- 不注入 production cutover flags。
- 不让 Go 执行 AI job。
- 不把 repo-owned temp smoke 误表述成 production runtime 已切换完成。
