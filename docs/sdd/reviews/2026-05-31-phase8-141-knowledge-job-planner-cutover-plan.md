# Phase 8.141 Review: Knowledge Job Planner Cutover Plan

## Scope

Added a read-only cutover plan for moving observe-only group memory/RAG job
admission from Python legacy enqueue to the Go `knowledge_job_planner`.

## Changes

- Added `KnowledgeJobPlannerCutoverPlanService`.
- Added `GET /v1/knowledge-job-planner/cutover-plan`.
- Added query/port/command types for the cutover plan.
- Wired the plan into `cmd/agent-runtime`.
- Aggregated the plan into `/v1/runtime-overview` summary/card/detail as
  `Knowledge Planner Cutover`.
- Stabilized `TestAgentJobCapacityPlanRecommendsRecoveryAndConcurrencyTuning`
  by using relative timestamps instead of a fixed wall-clock fixture.

## Boundary Check

Go owns only deterministic control-plane planning:

- current/desired/recommended admission owner;
- readiness blockers;
- operator enable, verification and rollback steps;
- runtime overview visibility.

Python still owns:

- executing `group_memory_extract` and `rag_ingest`;
- LLM/VLM/OCR/file understanding;
- Memory/RAG extraction, chunking, embedding, rerank and provider strategy;
- compatibility enqueue when the Go planner is disabled.

The endpoint does not mutate env/config, start workers, create AgentJob records,
lease work, call RAGFlow, or execute AI.

## Verification

- `go test ./app/service -run "TestKnowledgeJobPlannerCutoverPlan|TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics" -count=1 -v`
- `go test ./trigger/http -run TestKnowledgeJobPlannerCutoverPlanEndpointReturnsReadOnlyPlan -count=1 -v`
- `go test ./cmd/agent-runtime -run TestRuntimeConfig -count=1 -v`
- `go test ./...`

Result: all passing.

## Risks

- The plan is operational guidance only. Operators still need to apply env
  changes and restart services explicitly.
- Planner readiness proves admission conditions and Python worker liveness, not
  quality of extracted memory/RAG content.
