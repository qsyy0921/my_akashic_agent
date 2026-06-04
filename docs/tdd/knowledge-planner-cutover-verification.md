# TDD: Knowledge Planner Cutover Verification

## Targeted Coverage

1. Repo verification entrypoint:
   - read-only script can query runtime endpoints and summarize current cutover
     state
2. Live evidence surface:
   - recent planner buckets from runtime log
   - recent Go-created `group_memory_extract` jobs from runtime API
   - recent Python `skip legacy enqueue` signals from local log
3. Regression guard:
   - the script still works after future planner/runtime changes without adding
     new mutation behavior

## Commands

- `.\scripts\verify-knowledge-planner-cutover.ps1`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

## Deferred Coverage

- No automated unit test is added for the PowerShell log parsing in this slice;
  the verification script itself is exercised as a live runbook because the
  source of truth is the current runtime state plus current local logs.
