# Review: Dashboard Control Mutation Policy

Spec: `docs/sdd/specs/agent-gateway/100-dashboard-control-mutation-policy.md`

## Implementation Summary

- Added Python dashboard normalization for Go-owned `control_mutation_policy`.
- Added summary defaults for policy allowed/reason/targets/actions.
- Updated dashboard plugin test fixture and assertions for summary, card and
  top-level detail.

## Boundary Check

- Go remains the source of truth for mutation policy.
- Python dashboard only normalizes and exposes the Go overview payload.
- No approval creation, mutation audit creation, config/env mutation, cutover,
  worker startup, AgentJob mutation, MQ ack/nack, outbox dispatch or AI
  execution was added.

## Tests

- `uv run python -m py_compile plugins\runtime_overview\dashboard.py`
- `New-Item -ItemType Directory -Force .tmp\pytest | Out-Null; $env:TMP=(Resolve-Path .tmp\pytest).Path; $env:TEMP=$env:TMP; uv run pytest tests\test_runtime_overview_dashboard_plugin.py -q`

## Findings

- The dashboard can now consume policy detail from the same stable endpoint as
  control audit and other control-plane read models.
- No Go code changed in this slice.

## Decision

Accept.

## Follow-Ups

- Optional frontend table rendering can use `control_mutation_policy.intents`
  directly.
