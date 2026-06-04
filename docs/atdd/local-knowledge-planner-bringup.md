# ATDD: Local Knowledge Planner Bring-up

## Goal

Verify that the repo-local runtime entrypoint can bring up the Go
`knowledge_job_planner` and that recurring knowledge job admission moves from
Python legacy enqueue to Go without moving execution out of Python.

## Preconditions

- `services/agent-runtime` builds locally
- Python main process is already running with an active knowledge worker
- observe-only QQ group targets exist in Go runtime state

## Acceptance Checks

1. Start the local runtime with:
   `.\scripts\start-agent-runtime.ps1 -EnableKnowledgeJobPlanner -KnowledgeJobPlannerIntervalSeconds 300 -KnowledgeJobPlannerMaxAttempts 3 -KnowledgeJobPlannerRagMaxMessages 200`
2. Call `/v1/runtime-config` and confirm:
   - `workers.knowledge_job_planner_enabled=true`
   - `workers.outbox_delivery_worker_enabled=false`
3. Call `/v1/runtime-workers` and confirm:
   - `knowledge_job_planner.enabled=true`
   - `knowledge_job_planner.running=true`
4. Call `/v1/knowledge-job-planner/cutover-plan` and confirm:
   - `decision=ready`
   - `current_admission_owner=go_runtime_knowledge_job_planner`
5. Call `/v1/jobs?type=group_memory_extract` and confirm a fresh bucket exists
   with `metadata.scheduler=agent-runtime-knowledge-job-planner`
6. Wait past one legacy 60s enqueue interval and confirm the Python log does
   not emit a newer `agent_runtime_knowledge_worker] enqueued` line than the
   pre-cutover bucket.

## Failure Signals

- runtime script cannot enable the planner without manual file edits
- `knowledge_job_planner.running` stays false
- cutover plan still reports `python_legacy_knowledge_enqueue`
- Python continues creating new per-minute legacy enqueue buckets
