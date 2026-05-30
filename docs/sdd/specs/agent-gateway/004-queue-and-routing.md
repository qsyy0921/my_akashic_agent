# SPEC-004: Queue And Routing

## Status

Superseded for implementation sequencing by
`021-external-queue-backend.md`. The subject model remains the preferred NATS
JetStream direction, while implementation now starts with read-only
configuration diagnostics before enabling real external consumers.

## Context

Platform adapters, Agent reasoning, media handling, memory extraction, and audit
writing should be decoupled. A queue also provides retry and replay behavior.

## Decision

The original draft selected NATS JetStream. That remains the preferred MQ
because Akashic needs an event backbone, subject routing by platform/account/job
type, and bounded concurrent consumers. See
`021-external-queue-backend.md` for the migration sequence.

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

Each consumer group should use bounded Go goroutine worker pools. Concurrency is
configured with `AKASHIC_QUEUE_CONSUMER_CONCURRENCY`, while
`AKASHIC_QUEUE_MAX_IN_FLIGHT` prevents a single runtime from over-fetching work.

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
