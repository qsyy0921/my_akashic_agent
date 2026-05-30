# Review: Outbox Delivery Event Stream

Spec: `docs/sdd/specs/agent-gateway/020-outbox-event-stream.md`

## Implementation Summary

- Added `OutboxDeliveryEvent` domain model and validation.
- Added app query/view, assembler, in/out ports, and event viewer service.
- Added JSONL infrastructure adapter `infrastructure/outboxeventstore`.
- Extended memory store for in-process tests.
- Extended `MessageSendService` to record `queued` events when outbox delivery
  is created.
- Extended `OutboxService` to record `leased`, `dispatching`, `succeeded`,
  `failed`, and `retry` events.
- Added `GET /v1/outbox-events` and registered it in `agent-runtime`.
- Added runtime overview aggregation for recent outbox events.
- Added a shared contract fixture for an outbox delivery event stream.

## Tests Run

- `go test ./...`
- `uv run pytest tests/test_sdd_contract_fixtures.py -q --basetemp .tmp/pytest-contract-outbox-events`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q --basetemp .tmp/pytest-runtime-overview-outbox-events`

## Findings

- The event stream records action type and resulting status separately. This
  preserves dead-letter detail without inventing a separate `dead_lettered`
  event action.
- No platform sends are introduced by this slice.

## Decision

Approved. Outbox delivery state now has the same style of durable observability
as generic agent jobs.

## Follow-ups

- Decide whether the next queue backend should be NATS, Redis Streams, or
  another durable queue.
- Add a dedicated outbox-event detail view if operator workflow needs more than
  runtime overview aggregation.
