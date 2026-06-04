# TDD: Local Knowledge Planner Bring-up

## Targeted Coverage

1. Python knowledge worker compatibility logic:
   - existing tests already prove `enqueue_if_due_once()` suppresses legacy
     enqueue when runtime config reports
     `workers.knowledge_job_planner_enabled=true`
2. Regression checks for this slice:
   - Python knowledge worker still imports/runs after the new skip-log branch
   - governance/spec index checks still pass after adding the new SDD/ATDD/TDD
     records
3. Manual live verification:
   - repo-local runtime script can bring up the Go planner
   - runtime ownership flips in cutover-plan/runtime-overview
   - no new legacy enqueue records appear after enablement

## Commands

- `uv run pytest tests/test_agent_gateway_knowledge_worker.py -q`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

## Out of Scope

- Automated PowerShell entrypoint unit tests
- Python process restart automation to surface the new skip-log line in the
  already-running main process
