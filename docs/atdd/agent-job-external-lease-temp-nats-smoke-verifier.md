# Agent Job External Lease Temp NATS Smoke Verifier ATDD

## Scenario

operator 需要一个 repo-owned 入口，在不改动当前长期运行 runtime 的前提下，
直接证明 Go 侧 `agent_job result-ack` 的 duplicate-terminal-ack 和
pending/running/succeeded flow smoke 是可重复通过的。

## Acceptance

1. `.\scripts\verify-agent-job-external-lease-nats-smoke.ps1` 返回结构化 JSON。
2. 输出包含：
   - `docker.image`
   - `docker.mapped_port`
   - `go_test.selected_tests`
   - `checks`
   - `conclusion`
3. 当前本机运行后：
   - `checks.temp_nats_ready=true`
   - `checks.duplicate_terminal_ack_passed=true`
   - `checks.pending_running_succeeded_flow_passed=true`
   - `conclusion.status=live_verified`
   - `conclusion.category=repo_owned_temp_nats_smoke_live_verified`
4. `scripts/verify-go-migration-goal.ps1` 输出中新增
   `agent_job_external_lease_nats_smoke` 段，并让
   `residual_classification.agent_job_external_lease_result_ack_smoke`
   复用同一结论。

## Failure Signals

- 仍需要人工启动 NATS 并拼接 `go test` 命令
- smoke 因固定历史时间而误报 expired lease
- unified goal verifier 看不到 repo-owned temp NATS smoke 证据
