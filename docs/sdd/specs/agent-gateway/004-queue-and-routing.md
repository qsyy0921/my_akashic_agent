# SPEC-004: Queue And Routing

## Status

Draft

## Context

Platform adapters, Agent reasoning, media handling, memory extraction, and audit
writing should be decoupled. A queue also provides retry and replay behavior.

## Decision

Use NATS JetStream for the first gateway implementation.

## Subjects

```text
akashic.inbound.qq_1049511700
akashic.inbound.qq_2365524513
akashic.outbound.qq_1049511700
akashic.outbound.qq_2365524513
akashic.media.tasks
akashic.memory.group_ingest
akashic.audit.message
akashic.deadletter
```

## Consumers

- `agent-python`: consumes inbound subjects and emits outbound subjects.
- `media-worker`: consumes media tasks.
- `group-memory-worker`: consumes group ingest tasks.
- `audit-writer`: consumes audit events.

## Invariants

- All queue messages include `event_id`.
- Consumers must be idempotent by `event_id`.
- Failed messages retry with bounded attempts.
- Exhausted retries go to `akashic.deadletter`.
- Outbound routing must target exactly one concrete channel.

## Acceptance Tests

- Duplicate event ID is ignored by an idempotent consumer.
- Failed media task retries and then dead-letters.
- Outbound `qq_1049511700` is not sent by `qq_2365524513`.

