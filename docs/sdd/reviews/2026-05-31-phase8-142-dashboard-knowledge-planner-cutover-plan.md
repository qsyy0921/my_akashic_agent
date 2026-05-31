# Phase 8.142 Review: Dashboard Knowledge Planner Cutover Plan

## Scope

Expose the Go-owned knowledge job planner cutover plan through the Python
dashboard runtime overview read model.

## Changes

- Added dashboard summary defaults for `knowledge_job_planner_cutover_plan_*`.
- Normalized the top-level `knowledge_job_planner_cutover_plan` detail from Go
  runtime overview.
- Preserved the Go `Knowledge Planner Cutover` card as the operator-facing
  runtime overview card.
- Added dashboard plugin coverage for summary, card and detail fields.

## Boundary Check

Python dashboard remains a read-only projection:

- it does not decide admission ownership;
- it does not create `AgentJob` records;
- it does not mutate env/config;
- it does not start or stop workers;
- it does not execute group memory, RAG, OCR/VLM, image generation or any AI
  provider call.

Go remains the source of truth for the cutover plan and runtime overview
payload. Python only supplies stable defaults and shape normalization for the
existing dashboard endpoint.

## Verification

- `uv run pytest --basetemp .\.tmp\pytest tests/test_runtime_overview_dashboard_plugin.py -q`

Result: 5 passed.

## Risks

- This only improves dashboard visibility. It does not enable the Go
  `knowledge_job_planner`; operators still need to apply the documented env
  change and restart/verify services explicitly.
