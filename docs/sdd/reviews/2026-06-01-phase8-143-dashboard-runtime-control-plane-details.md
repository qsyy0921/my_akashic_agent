# Phase 8.143 Review: Dashboard Runtime Control Plane Details

## Scope

Expose additional Go-owned runtime overview control-plane details through the
Python dashboard read model.

## Changes

- Added dashboard summary defaults for:
  - `agent_job_capacity_*`;
  - `agent_job_external_lease_*`;
  - `agent_job_external_lease_plan_*`;
  - `outbound_cutover_plan_*`.
- Normalized top-level dashboard details:
  - `agent_job_capacity_plan`;
  - `agent_job_external_lease_readiness`;
  - `agent_job_external_lease_plan`;
  - `outbound_cutover_plan`.
- Kept Go runtime overview cards as the card source of truth.
- Added dashboard plugin assertions for summary, card and detail fields.

## Boundary Check

Python dashboard remains a read-only projection. It does not:

- create, lease, retry or ack/nack `AgentJob`;
- create or dispatch outbox deliveries;
- mutate env/config;
- start or stop workers;
- call QQ/Telegram adapters;
- execute model, Memory/RAG, OCR/VLM, image generation or tool work.

Go remains the source of truth for capacity, external lease and cutover
decisions. Python only stabilizes the dashboard API shape.

## Verification

- `uv run pytest --basetemp .\.tmp\pytest tests/test_runtime_overview_dashboard_plugin.py -q`
- `uv run python -m py_compile plugins/runtime_overview/dashboard.py tests/test_runtime_overview_dashboard_plugin.py`

Result: both passed.

## Risks

- This improves operator visibility only. Real outbox or AgentJob cutover still
  requires the existing Go readiness/plan gates, live smoke and explicit
  operator configuration changes.
