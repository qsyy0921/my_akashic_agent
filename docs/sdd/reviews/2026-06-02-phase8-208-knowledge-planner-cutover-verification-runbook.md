# Phase 8.208 Review: knowledge planner cutover verification runbook

## What changed

- Added a repo-owned read-only verification script:
  - `scripts/verify-knowledge-planner-cutover.ps1`
- The script re-checks live runtime state and recent local logs to verify that
  Go owns knowledge admission while Python only executes leased work.

## What was verified

- `.\scripts\verify-knowledge-planner-cutover.ps1` ran successfully against the
  current runtime and returned JSON.
- The current runtime reported:
  - planner enabled and running
  - current admission owner
    `go_runtime_knowledge_job_planner`
  - recent planner buckets through bucket `5934697`
  - recent `group_memory_extract` jobs tagged with
    `scheduler=agent-runtime-knowledge-job-planner`
  - recent Python `skip legacy enqueue` signals plus processed Go-created jobs
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
  passed.

## Outcome

Knowledge planner long-running verification no longer depends on manual
cross-checking of multiple endpoints and logs. The goal still remains active
because Telegram backend is still missing token-backed smoke, and QQ rich-media
image / first-account group-file blockers still exist.
