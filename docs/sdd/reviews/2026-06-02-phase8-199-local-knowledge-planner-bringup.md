# Review: local knowledge planner bring-up

Spec:
- `docs/sdd/specs/agent-gateway/142-local-knowledge-planner-bringup.md`

Implementation summary:
- Extended `scripts/start-agent-runtime.ps1` so repo-local runtime bring-up can
  explicitly enable `knowledge_job_planner` and pass interval/max-attempts/rag
  knobs without hand-editing the script.
- Added an explicit Python log branch in
  `integrations/agent_gateway_knowledge_worker.py` for future
  `skip legacy enqueue` evidence when Go owns recurring admission.
- Re-enabled the local runtime with the Go planner and verified the ownership
  flip through runtime endpoints and fresh `group_memory_extract` jobs.

Tests run:
- live `start-agent-runtime.ps1 -EnableKnowledgeJobPlanner ...`
- `/v1/runtime-config`
- `/v1/runtime-workers`
- `/v1/knowledge-job-planner/readiness`
- `/v1/knowledge-job-planner/cutover-plan`
- `/v1/jobs?type=group_memory_extract`

Findings:
- The first script revision exposed a PowerShell syntax issue caused by using
  `if` directly in hashtable value expressions; fixed by precomputing planner
  values before building the env block.
- After enabling the planner, Go created a fresh bucket of 6
  `group_memory_extract` jobs with `scheduler=agent-runtime-knowledge-job-planner`.
- The Python log stopped producing new per-minute legacy enqueue entries after
  planner enablement, which is consistent with runtime-config-based suppression.

Decision:
- Accept. Local knowledge admission can now be brought up repeatably through the
  repo script, and the runtime owner has been advanced to Go for this machine.

Follow-ups:
- Re-run dashboard `/api/dashboard/runtime-overview` knowledge planner cutover
  drilldown under the new runtime state.
- After the next Python main-process restart, verify the new
  `skip legacy enqueue` log line appears directly.
