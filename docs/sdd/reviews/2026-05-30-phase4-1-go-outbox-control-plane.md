# Review: Phase 4.1 Go Outbox Control Plane

Spec:

- `docs/sdd/specs/agent-gateway/006-outbox-delivery-retry.md`
- `docs/sdd/specs/agent-architecture/005-migration-plan.md`
- `docs/sdd/specs/agent-architecture/008-go-migration-scope.md`

Implementation summary:

- Added Go `OutboxDelivery` domain lifecycle with queued, dispatching,
  succeeded, failed, and dead-lettered states.
- Added app-layer outbox service and ports following the existing DDD +
  hexagonal package rules.
- Updated `/v1/outbound` to create an outbox delivery and enqueue it in the Go
  in-memory infrastructure store.
- Added `/v1/outbox` query and state transition HTTP endpoints.
- Kept actual platform SDK sends on the Python compatibility path; this is not
  a production sender cutover.

Tests run:

- `gofmt -w api app cmd domain infrastructure trigger types`
- `go test ./...`

Findings:

- The current store is still in-memory. This proves the domain/app/API boundary,
  but does not yet satisfy durable restart recovery.
- Dashboard does not yet render outbox state; the API is ready for a dashboard
  panel in a later slice.
- Platform delivery adapters remain outside this change by design.

Decision:

- Accept as a Phase 4.1 control-plane migration slice.

Follow-ups:

- Add JSONL or SQLite persistence for outbox deliveries.
- Add dashboard outbox/error view.
- Add Go platform delivery adapters behind an outbound port.
- Add a Python compatibility client that reports actual send success/failure to
  `/v1/outbox/{event_id}/succeeded|failed`.
