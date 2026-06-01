# Runtime Overview Queue Topology

## Context

`/v1/queue-topology` now exposes a Go-owned read-only topology view for queue
provider state, execution owners, ack owners, nodes, edges and blocked work
kinds. Operators should not need to call a separate endpoint to understand the
current queue boundary when they already use `/v1/runtime-overview` and the
Python dashboard as the main status surface.

This slice aggregates queue topology into runtime overview and dashboard.

## Boundary

Go owns:

- queue topology derivation;
- runtime overview summary/card/detail aggregation;
- execution owner and ack owner visibility;
- external lease gate and blocker counts.

Python dashboard owns only:

- stable read-model defaults;
- shape normalization for browser/API consumers;
- preserving Go cards and details as read-only data.

Neither Go runtime overview nor Python dashboard may:

- publish, lease, ack, nack or term queue work;
- create AgentJob or outbox records;
- mutate env/config;
- start or stop workers;
- send QQ/Telegram messages;
- call AI providers or execute Memory/RAG/OCR/VLM/image work.

## Design

Extend `RuntimeOverviewDeps` with `QueueTopology`.

Runtime overview adds:

- top-level `queue_topology`;
- summary fields:
  - `queue_topology_nodes`;
  - `queue_topology_edges`;
  - `queue_topology_work_kinds`;
  - `queue_topology_blockers`;
  - `queue_topology_external_lease_ready`;
  - `queue_topology_outbox_execution_owner`;
  - `queue_topology_agent_job_execution_owner`;
  - `queue_topology_agent_job_ack_owner`;
- card:
  - `Queue Topology`.

Python dashboard normalizes the same top-level detail and adds summary defaults.

## Verification

- `go test ./app/service -run TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics -count=1 -v`
- `go test ./cmd/agent-runtime -run TestRuntimeConfig -count=1 -v`
- `go test ./...`
- `uv run pytest --basetemp .\.tmp\pytest tests/test_runtime_overview_dashboard_plugin.py -q`
- `uv run python -m py_compile plugins/runtime_overview/dashboard.py tests/test_runtime_overview_dashboard_plugin.py`

## Risks

- Runtime overview only reflects current Go control-plane state. It does not
  prove NATS connectivity or worker liveness beyond the existing diagnostic
  fields.
- Dashboard must not grow actions from this card without a separate
  operator-ack design.
