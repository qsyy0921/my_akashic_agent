# Review: Phase 8.55 Agent Job NATS Scope Gate

## Scope

- Added an explicit gate for `agent_job` NATS external-lease subject
  consumption.
- Added a NATS-level duplicate terminal delivery smoke for generic jobs.
- This still does not let Go execute Python-owned model, RAG, OCR/VLM, or
  memory work.

## Design

- The external lease consumer subscribes to `{subject_prefix}.outbox.>` by
  default.
- When `agent_job` result-ack is explicitly enabled, the subscription expands to
  `{subject_prefix}.>` so outbox and generic job notifications can both reach
  the same bounded worker pool.
- The default durable changes from `AKASHIC_EXTERNAL_LEASE_OUTBOX` to
  `AKASHIC_EXTERNAL_LEASE_ALL` for the expanded subscription, avoiding durable
  filter conflicts with an existing outbox-only consumer.

## Gate

`agent_job` result-ack scope requires all base external-lease gates plus:

- `AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED=true`
- `AKASHIC_QUEUE_AGENT_JOB_DUPLICATE_SMOKE_PASSED=true`
- `AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN=true`

When enabled, `/v1/queue-backend` reports:

- `execution_scope=outbox_delivery_and_agent_job_result_ack`
- `allowed_work_kinds=["outbox_delivery","agent_job"]`
- `agent_job_queue_source=agent_job_state_store_with_nats_result_ack`

## Verification

- `go test ./infrastructure/natsqueue ./cmd/agent-runtime ./smoke -count=1`
- `AKASHIC_NATS_SMOKE_DSN=nats://127.0.0.1:4222 go test ./smoke -run "TestExternalLeaseNATSSmoke(OutboxDispositions|AgentJobDuplicateTerminalAck)" -count=1 -v`

The second command was run against a temporary local `nats:2-alpine -js`
container. It does not send QQ/Telegram messages and does not invoke Python
workers.

## Remaining Risk

- A real runtime dry-run is still needed before setting the new scope flags in
  the user's long-running local agent process.
- Startup/background expired-lease recovery remains a separate design decision.
