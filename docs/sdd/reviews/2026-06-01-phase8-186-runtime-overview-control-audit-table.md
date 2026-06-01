# Review: Runtime Overview Control Audit Table

## Scope

- Render the Go-owned `control_audit` runtime overview detail as read-only
  dashboard tables.
- Preserve raw JSON fallback/debug output.

## Result

- The runtime overview panel detects card id `control_audit` and renders
  operator approval totals plus a bounded approvals table.
- The same detail renders control mutation totals plus a bounded mutations
  table.
- The renderer consumes only the already loaded card detail and makes no
  network requests.

## Boundary

Go remains the source of truth for operator approval and control mutation audit
state. The dashboard only presents data. It does not create, check, revoke, or
approve operator approvals; does not create planned/applied/failed mutation
records; does not execute cutover, worker scaling, queue ack/nack, media
cleanup, or AI work.

## Tests

- `npm run build:plugins`
- `npm run typecheck`
- `go test ./...`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `git diff --check`
