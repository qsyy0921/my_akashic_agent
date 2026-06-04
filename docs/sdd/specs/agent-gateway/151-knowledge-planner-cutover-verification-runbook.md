# Spec 151: knowledge planner cutover verification runbook

## Context

The current Go `knowledge_job_planner` is already the admission owner in live
runtime, but verification has still depended on ad hoc endpoint reads and
manual log inspection.

That is error-prone because the cutover claim depends on multiple live signals
at once:

- runtime config says planner is enabled
- runtime workers say planner is running
- readiness says Python knowledge worker is healthy
- cutover plan says current owner is Go
- recent `group_memory_extract` jobs were created by
  `agent-runtime-knowledge-job-planner`
- Python logs show `skip legacy enqueue` while still processing Go-created jobs

## Goal

Provide a repo-owned verification entrypoint that checks the live knowledge
planner cutover from current runtime state and recent logs, so each iteration
can re-prove the same boundary without reassembling evidence by hand.

## Requirements

1. Add a repo-owned script:
   - `scripts/verify-knowledge-planner-cutover.ps1`
2. The script must read live runtime state from:
   - `/v1/runtime-config`
   - `/v1/runtime-workers`
   - `/v1/knowledge-job-planner/readiness`
   - `/v1/knowledge-job-planner/cutover-plan`
   - `/v1/jobs?type=group_memory_extract`
3. The script must also read recent local evidence from:
   - `logs/agent-runtime-local.err.log`
   - `logs/akashic-local-run.err.log`
4. The script must emit machine-readable JSON that makes these checks explicit:
   - planner enabled and running
   - current admission owner is `go_runtime_knowledge_job_planner`
   - recent planner buckets exist in runtime log
   - recent `group_memory_extract` jobs were created by
     `agent-runtime-knowledge-job-planner`
   - recent Python `skip legacy enqueue` signals exist
5. The script must remain read-only:
   - no job creation
   - no env mutation
   - no process restart
   - no QQ/Telegram send

## Non-Goals

- Do not change knowledge admission ownership in this slice.
- Do not add new runtime APIs just for the script.
- Do not move knowledge execution out of Python.

## Acceptance

- Running `.\scripts\verify-knowledge-planner-cutover.ps1` returns JSON.
- The JSON includes:
  - current runtime planner state
  - recent runtime planner buckets
  - recent Go-created `group_memory_extract` jobs
  - recent Python `skip legacy enqueue` and processed-job signals
  - explicit boolean checks for the live cutover claim
- The current live runtime shows:
  - `planner_running_and_ready=true`
  - `recent_planner_buckets_seen=true`
  - `recent_go_scheduler_jobs_seen=true`
  - `recent_skip_legacy_enqueue_seen=true`
