# ATDD: Knowledge Planner Cutover Verification

## Goal

Verify from current live runtime and current logs that Go owns recurring
knowledge-job admission while Python only executes the leased jobs.

## Preconditions

- `agent-runtime` is healthy on `127.0.0.1:8780`
- `knowledge_job_planner` is enabled in current runtime
- Python main process is running
- `logs/agent-runtime-local.err.log` and `logs/akashic-local-run.err.log`
  exist in the repo

## Acceptance Checks

1. Run:
   - `.\scripts\verify-knowledge-planner-cutover.ps1`
2. Confirm the JSON shows:
   - `runtime.knowledge_job_planner_enabled=true`
   - `runtime.planner_worker_running=true`
   - `cutover_plan.current_admission_owner=go_runtime_knowledge_job_planner`
3. Confirm `recent_planner_buckets` contains recent bucket entries from
   `agent-runtime-local.err.log`.
4. Confirm `recent_group_memory_jobs` contains jobs with:
   - `scheduler=agent-runtime-knowledge-job-planner`
   - `status=succeeded`
5. Confirm `python_signals.skip_legacy_enqueue` contains recent
   `go_runtime_knowledge_job_planner_enabled` lines.
6. Confirm the summary checks are all true:
   - `planner_running_and_ready`
   - `recent_planner_buckets_seen`
   - `recent_go_scheduler_jobs_seen`
   - `recent_skip_legacy_enqueue_seen`

## Failure Signals

- script cannot read live runtime endpoints
- planner is enabled but not running
- current admission owner is not Go
- recent jobs are no longer tagged with
  `agent-runtime-knowledge-job-planner`
- Python logs stop showing `skip legacy enqueue`
