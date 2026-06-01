# Dashboard Runtime Control Plane Details

## Context

Go `agent-runtime` already aggregates several read-only control-plane views into
`/v1/runtime-overview`:

- `agent_job_capacity_plan`;
- `agent_job_external_lease_readiness`;
- `agent_job_external_lease_plan`;
- `outbound_cutover_plan`.

The Python dashboard currently keeps the Go cards, but does not provide stable
top-level normalized details or default summary fields for all of these views.
That makes the browser/API consumer rely on raw Go payload shapes and weakens
the dashboard as the stable operator API.

## Boundary

Go owns deterministic control-plane state:

- AgentJob pressure, capacity recommendations and worker coverage;
- generic job external lease result-ack readiness and cutover plan;
- outbound execution owner cutover plan;
- blocker counts, step lists, execution/admission owner fields and side-effect
  declarations.

Python dashboard owns only read-model normalization:

- fill missing summary defaults;
- normalize read-only detail objects;
- preserve Go cards;
- keep the existing `/api/dashboard/runtime-overview` endpoint shape stable.

Python dashboard must not:

- create, lease, retry or ack/nack `AgentJob`;
- create or dispatch outbox deliveries;
- modify env/config;
- start or stop Go/Python workers;
- call QQ/Telegram adapters;
- execute model, Memory/RAG, OCR/VLM or image generation work.

## Design

Extend `_normalize_go_runtime_overview` to normalize:

- `agent_job_capacity_plan`;
- `agent_job_external_lease_readiness`;
- `agent_job_external_lease_plan`;
- `outbound_cutover_plan`.

Add summary defaults for the Go runtime overview fields that already exist in
Go:

- `agent_job_capacity_*`;
- `agent_job_external_lease_*`;
- `agent_job_external_lease_plan_*`;
- `outbound_cutover_plan_*`.

Use shared small helpers where possible:

- normalized blocker/note string lists;
- normalized operator steps with `step_index`, `phase`, `action`, `detail`,
  `method`, `endpoint`, `env`;
- compact readiness summaries nested under plan views.

The implementation is a dashboard projection only. Cards remain the Go-provided
cards from runtime overview.

## Verification

- `uv run pytest --basetemp .\.tmp\pytest tests/test_runtime_overview_dashboard_plugin.py -q`
- `uv run python -m py_compile plugins/runtime_overview/dashboard.py tests/test_runtime_overview_dashboard_plugin.py`

## Risks

- This does not perform real QQ/NapCat outbound or NATS cutover. It only makes
  the operator dashboard consume the already-Go-owned control-plane views.
- The dashboard normalizers intentionally keep provider-specific raw nested
  detail small. Deeper UIs should still consume Go detail fields, not re-create
  control logic in Python.
