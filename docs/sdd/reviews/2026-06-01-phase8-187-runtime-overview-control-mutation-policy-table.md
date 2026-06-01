# Review: Runtime Overview Control Mutation Policy Table

## Scope

- Render the Go-owned `control_mutation_policy` runtime overview detail as a
  read-only dashboard table.
- Preserve raw JSON fallback/debug output.

## Result

- The runtime overview panel detects card id `control_mutation_policy` and
  renders allowed/reason/target/action totals.
- The detail shows a bounded target-kind to allowed-actions table.
- The renderer consumes only the already loaded card detail and makes no
  network requests.

## Boundary

Go remains the source of truth for the control mutation policy allowlist and
preflight behavior. The dashboard only presents data. It does not edit policy
entries, create approvals or mutation audits, execute cutover, start workers,
ack/nack MQ, run media cleanup, or invoke AI work.

## Tests

- `npm run build:plugins`
- `npm run typecheck`
- `go test ./...`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `git diff --check`
