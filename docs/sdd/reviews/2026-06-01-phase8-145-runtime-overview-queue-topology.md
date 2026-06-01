# Phase 8.145 Review: Runtime Overview Queue Topology

## Scope

Aggregated the Go-owned queue topology read model into runtime overview and the
Python dashboard runtime overview projection.

## Changes

- Added `QueueTopology` to `RuntimeOverviewDeps`.
- Added top-level `queue_topology` to `RuntimeOverviewView`.
- Added queue topology summary fields for nodes, edges, work kinds, blockers,
  external lease readiness, outbox execution owner, AgentJob execution owner and
  AgentJob ack owner.
- Added `Queue Topology` runtime overview card.
- Wired `QueueTopologyService` into runtime overview in `cmd/agent-runtime`.
- Added Python dashboard normalization for `queue_topology`.
- Extended Go and dashboard tests.

## Boundary Check

Go remains the source of truth for queue topology and runtime overview
aggregation. Python dashboard only normalizes the read model.

No code in this slice publishes, leases, ack/nacks, terms, creates AgentJobs,
creates outbox deliveries, sends platform messages, mutates env/config, starts
workers, or executes AI/model/RAG/OCR/VLM/image work.

## Verification

- `go test ./app/service -run TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics -count=1 -v`
- `go test ./cmd/agent-runtime -run TestRuntimeConfig -count=1 -v`
- `go test ./...`
- `uv run pytest --basetemp .\.tmp\pytest tests/test_runtime_overview_dashboard_plugin.py -q`
- `uv run python -m py_compile plugins/runtime_overview/dashboard.py tests/test_runtime_overview_dashboard_plugin.py`

Result: all passing.

## Risks

- The topology remains a read-only control-plane view. It does not prove live
  NATS connectivity or platform delivery quality.
- A future UI panel should remain read-only unless a separate operator-ack and
  audit design is added.
