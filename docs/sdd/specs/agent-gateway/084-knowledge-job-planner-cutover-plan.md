# Knowledge Job Planner Cutover Plan

## Context

Go already owns observe-only knowledge job admission through
`knowledge_job_planner`, and exposes read-only preview/readiness endpoints.
Operators can see whether planned `group_memory_extract` / `rag_ingest` jobs
exist and whether Python knowledge workers are available, but the enable and
rollback procedure is still scattered across notes and env names.

This slice adds a read-only cutover plan for knowledge job admission. It is a
control-plane plan, not a mutation endpoint.

## Boundary

Go owns deterministic admission control-plane state:

- observe-only target enumeration;
- planned `group_memory_extract` and `rag_ingest` AgentJob admission;
- planner readiness;
- runtime worker status and Python worker liveness as control-plane signals;
- operator-facing enable, verification and rollback instructions.

Python remains responsible for AI work:

- executing leased `group_memory_extract` / `rag_ingest` jobs;
- LLM/VLM/OCR/file understanding;
- Memory/RAG extraction, chunking, embedding, rerank and provider strategy;
- legacy enqueue compatibility when Go planner is disabled.

The new plan must not:

- create AgentJob records;
- mutate env/config;
- start or stop runtime workers;
- call Python workers or model providers;
- execute RAGFlow, OCR, VLM, image generation or memory extraction.

## Design

Add a Go application service:

```text
KnowledgeJobPlannerCutoverPlanService
  -> KnowledgeJobPlannerReadinessChecker
  -> KnowledgeJobPlannerCutoverPlanView
```

The view reports:

- `decision`;
- current admission owner:
  - `go_runtime_knowledge_job_planner`
  - `python_legacy_knowledge_enqueue`
- desired and recommended admission owner;
- embedded readiness view;
- blockers;
- required checks, enable steps, verification steps and rollback steps;
- `side_effect=none`.

Rules:

- recommended owner is `go_runtime_knowledge_job_planner`;
- current owner is derived from readiness planner state:
  - enabled and running -> Go runtime planner;
  - otherwise -> Python legacy enqueue;
- desired owner defaults to recommended;
- Go desired owner requires readiness `ready=true`;
- Python legacy owner is always a rollback/fallback owner;
- `ready=true` only means current owner already equals desired owner and there
  are no blockers.

Expose:

- `GET /v1/knowledge-job-planner/cutover-plan`
- runtime overview summary/card/detail:
  - `knowledge_job_planner_cutover_plan_ready`
  - `knowledge_job_planner_cutover_plan_decision`
  - `knowledge_job_planner_cutover_plan_blockers`
  - `Knowledge Planner Cutover`

## Verification

- `go test ./app/service -run TestKnowledgeJobPlannerCutoverPlan -count=1 -v`
- `go test ./trigger/http -run TestKnowledgeJobPlannerCutoverPlanEndpointReturnsReadOnlyPlan -count=1 -v`
- `go test ./app/service -run TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics -count=1 -v`
- `go test ./cmd/agent-runtime -run TestRuntimeConfig -count=1 -v`

## Risks

- This plan does not prove Python knowledge extraction quality. It only checks
  admission readiness and worker availability.
- The endpoint intentionally does not flip `AKASHIC_KNOWLEDGE_JOB_PLANNER_ENABLED`;
  operators still need to apply env changes and restart services explicitly.
