# Dashboard Knowledge Planner Cutover Plan

## Context

Go runtime now exposes a read-only
`/v1/knowledge-job-planner/cutover-plan` and aggregates it into
`/v1/runtime-overview` as `knowledge_job_planner_cutover_plan`.

The Python dashboard already prefers `/v1/runtime-overview`, but its
normalization layer predates this detail block. Without explicit normalization,
the frontend cannot rely on stable summary defaults or a top-level detail object
for operator drilldown.

## Boundary

Go owns the deterministic control-plane state:

- knowledge planner cutover decision;
- current/desired/recommended admission owner;
- blockers and operator steps;
- runtime overview summary/card/detail payloads.

Python dashboard owns only presentation/read adaptation:

- normalizing Go JSON into the dashboard contract;
- supplying default zero/empty values for missing summary keys;
- preserving fallback behavior when Go aggregate is unavailable.

Python dashboard must not:

- create AgentJob records;
- mutate runtime config or environment variables;
- start or stop workers;
- run group memory/RAG extraction;
- call LLM/VLM/OCR/file parsing/image generation providers.

## Design

Extend `plugins/runtime_overview/dashboard.py`:

1. Add summary defaults for:
   - `knowledge_job_planner_cutover_plan_ready`
   - `knowledge_job_planner_cutover_plan_decision`
   - `knowledge_job_planner_cutover_plan_blockers`
   - `knowledge_job_planner_cutover_plan_current_owner`
   - `knowledge_job_planner_cutover_plan_desired_owner`
   - `knowledge_job_planner_cutover_plan_recommended_owner`
2. Normalize `knowledge_job_planner_cutover_plan` as a read-only detail:
   - decision and owners;
   - embedded readiness summary;
   - required/enable/verification/rollback steps;
   - blockers, attributes, notes and `side_effect`.
3. Return the detail from `_normalize_go_runtime_overview`.

The dashboard does not add a separate endpoint in this slice; it consumes the Go
aggregate detail already present in `/v1/runtime-overview`.

## Verification

- `uv run pytest --basetemp .\.tmp\pytest tests/test_runtime_overview_dashboard_plugin.py -q`

## Risks

- This exposes operational guidance only. Operators still need to apply env
  changes explicitly.
- The plan proves admission readiness and worker liveness, not extracted memory
  or RAG answer quality.
