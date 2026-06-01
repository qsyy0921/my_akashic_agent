# Review: Dashboard Control Audit Detail

Spec: `docs/sdd/specs/agent-gateway/095-dashboard-control-audit-detail.md`

## Implementation Summary

- Python dashboard runtime overview now normalizes Go-owned `operator_approvals`
  and `control_mutations`.
- Summary defaults now include operator approval and control mutation counters.
- Dashboard plugin tests cover the `control_audit` card, summary counters and
  approval/mutation detail fields.
- The dashboard remains a read-only display adapter: it does not create
  approvals, record mutations, run approval checks, change config, execute
  cutover, create or lease AgentJobs, ack/nack MQ, or invoke AI.

## Tests

- `uv run pytest tests\test_runtime_overview_dashboard_plugin.py -q`
  - Initial run failed because the default Windows pytest temp directory was not
    writable: `PermissionError: [WinError 5]`.
- `New-Item -ItemType Directory -Force .tmp\pytest | Out-Null; $env:TMP=(Resolve-Path .tmp\pytest).Path; $env:TEMP=$env:TMP; uv run pytest tests\test_runtime_overview_dashboard_plugin.py -q`
  - Passed: `5 passed`.

## Findings

- The implementation stays within the intended boundary: Go owns the ledgers
  and runtime overview aggregation; Python only normalizes fields for the
  dashboard API.
- No Go code changed in this slice.

## Decision

Accept.

## Follow-Ups

- Optional frontend drilldown can render tables from the normalized detail.
- Do not add real control-plane mutation logic to the dashboard.
