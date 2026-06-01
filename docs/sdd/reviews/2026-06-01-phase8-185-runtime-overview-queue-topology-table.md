# Review: Runtime Overview Queue Topology Table

## Scope

- Render the Go-owned `queue_topology` runtime overview detail as a read-only
  dashboard table.
- Preserve raw JSON fallback/debug output.

## Result

- The runtime overview panel detects card id `queue_topology` and renders
  provider/mode/phase/external lease summary, node/edge counts, and a work-kind
  table.
- Each work-kind row shows queue source, execution owner, ack owner, allowed
  state and blockers.
- The renderer consumes only the already loaded card detail and makes no network
  requests.

## Boundary

Go remains the source of truth for queue topology and execution boundaries.
The dashboard only presents the data. It does not publish, lease, ack, nack, or
term MQ messages; does not create, retry, or execute AgentJobs/outbox
deliveries; does not start workers or trigger Python AI.

## Tests

- `npm run build:plugins`
- `npm run typecheck`
- `go test ./...`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `git diff --check`
