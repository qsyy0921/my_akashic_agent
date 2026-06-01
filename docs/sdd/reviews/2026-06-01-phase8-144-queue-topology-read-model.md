# Phase 8.144 Review: Queue Topology Read Model

## Scope

Added a Go-owned read-only queue topology view for operator/API visibility.

## Changes

- Added `QueueTopologyView` query types.
- Added `QueueTopologyViewer` input port.
- Added `QueueTopologyService` in the app layer, deriving topology from
  `QueueBackendViewer`.
- Added `GET /v1/queue-topology`.
- Wired the endpoint in `cmd/agent-runtime`.
- Added service and HTTP contract tests.

## Boundary Check

Go owns the deterministic control-plane projection:

- provider, mode and migration phase;
- selected/recommended provider;
- state-store vs external queue nodes;
- `outbox_delivery` and `agent_job` work-kind ownership;
- execution owner, ack owner, external lease gate state and blockers.

Python remains responsible for AI work:

- model/RAG/OCR/VLM/image workers;
- prompt/tool orchestration;
- provider-specific retry/fallback;
- external lease result-ack client behavior when enabled.

The endpoint is read-only. It does not publish, lease, ack, nack, term, create
jobs, create deliveries, send platform messages, mutate env/config, or execute
AI work.

## Verification

- `go test ./app/service -run TestQueueTopology -count=1 -v`
- `go test ./trigger/http -run TestQueueTopologyEndpointReturnsReadOnlyView -count=1 -v`
- `go test ./...`

Result: all passing.

## Risks

- The topology reflects configured Go control-plane state and diagnostics. It
  does not prove live NATS connectivity or platform delivery health.
- The next frontend step should consume `/v1/queue-topology` as a read-only
  panel; it should not introduce dashboard-side cutover actions without a
  separate operator-ack design.
