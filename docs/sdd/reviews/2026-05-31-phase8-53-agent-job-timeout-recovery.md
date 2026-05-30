# Review: Phase 8.53 Agent Job Timeout Recovery

## Scope

- Added Go-owned expired lease recovery for generic `AgentJob` work.
- Recovery remains a manual/control-plane operation; it does not start live QQ,
  Telegram, Python worker, or NATS side effects.
- Python remains responsible for model/RAG/memory execution, while Go owns
  deterministic lifecycle state, fencing, and timeout recovery.

## Design

- Domain: `AgentJob.RecoverExpiredLease` is the only place that decides timeout
  state transitions.
- Repository: `ListExpiredAgentJobLeases` provides a bounded scan of
  `leased` / `running` jobs whose lease is expired.
- App service: `RecoverExpiredLeases` saves recovered state and emits
  `lease_expired` events.
- HTTP/client: `POST /v1/jobs/recover-expired` and Python
  `AgentGatewayClient.recover_expired_jobs` are thin control-plane wrappers.

## Behavior

- Retryable expired jobs return to `pending` with lease owner, lease token, and
  lease expiry cleared.
- Exhausted expired jobs move to `dead_lettered` with a timeout reason.
- Recovery summary includes scanned, recovered, dead-lettered, and per-job
  action data.

## Verification

- `go test ./domain/model ./app/service ./infrastructure/agentjobstore ./trigger/http`
- `go test ./...`
- `go build ./cmd/agent-runtime`
- `uv run pytest tests\test_agent_gateway_client.py -q --basetemp .tmp\pytest-recover-expired`

## Remaining Risk

- Recovery is not yet wired into the NATS external lease consumer loop.
- NATS result acknowledgement still needs duplicate-delivery handling and
  explicit `ack` / delayed `nack` / `term` mapping for `agent_job` work.
