# 142 Local Knowledge Planner Bring-up

## Context

The Go `knowledge_job_planner` already owns preview, readiness, cutover-plan,
and the runtime worker implementation. But the repo-local start entrypoint still
hard-coded a planner-disabled runtime, which made the final cutover check a
manual environment-edit exercise instead of a repeatable local operation.

At the same time, Python already knew how to suppress legacy enqueue when Go
advertised planner ownership, but the running logs did not make that suppression
explicit enough for quick operator verification.

## Decision

1. Extend `scripts/start-agent-runtime.ps1` so local bring-up can explicitly
   enable the Go knowledge planner and pass the bounded planner knobs used by
   the existing cutover plan:
   - enable flag
   - interval seconds
   - max attempts
   - rag max messages
   - run-on-start
2. Keep outbox execution owner unchanged. This slice only brings up the Go
   knowledge admission owner.
3. Make Python `AgentGatewayKnowledgeWorker` log a clear
   `skip legacy enqueue` line when Go owns recurring admission.

## Boundary

### Go owns

- recurring observe-only knowledge job admission
- local runtime worker bring-up for `knowledge_job_planner`
- planner runtime diagnostics and cutover owner state

### Python owns

- leasing and executing `group_memory_extract` / `rag_ingest`
- group memory extraction, RAG ingest, prompt/model behavior
- compatibility fallback when Go planner is disabled or unavailable

## Non-goals

- No migration of knowledge execution into Go
- No default enabling of the outbox delivery worker
- No change to Telegram, scheduler, proactive, or media recovery ownership
- No forced restart of the Python main process as part of the script

## Validation

- Local runtime can be started with Go planner enabled from the repo script
- `/v1/runtime-config` reports `knowledge_job_planner_enabled=true`
- `/v1/runtime-workers` reports `knowledge_job_planner.running=true`
- `/v1/knowledge-job-planner/cutover-plan` flips current owner to
  `go_runtime_knowledge_job_planner`
- Python stops producing new legacy per-minute enqueue records while still
  executing Go-created knowledge jobs
