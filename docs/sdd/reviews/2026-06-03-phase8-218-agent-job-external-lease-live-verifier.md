# Review: agent job external lease live verifier

Spec:
- `docs/sdd/specs/agent-gateway/161-agent-job-external-lease-live-verifier.md`

Implementation summary:
- Added `scripts/verify-agent-job-external-lease.ps1` to aggregate current
  readiness, plan, runtime flags, queue backend, queue topology, runtime
  overview summary, and Python worker coverage into one live report.
- Updated `scripts/verify-go-migration-goal.ps1` to embed the new verifier
  output and reuse its conclusion for
  `residual_classification.agent_job_external_lease_result_ack`.

Tests run:
- `.\scripts\verify-agent-job-external-lease.ps1`
- `.\scripts\verify-go-migration-goal.ps1`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

Findings:
- Current local runtime is not blocked by Python worker coverage.
- Current blocker is configuration and cutover flags: provider still `local`,
  external queue is not configured, strict lease token is off, result-ack flag
  is off, and duplicate/flow smoke flags are missing.
- Queue topology and runtime overview both still agree that `agent_job`
  execution owner is `python_ai_worker_state_store_lease`.

Decision:
- Accept.

Follow-ups:
- When NATS credentials and cutover intent are available, use this verifier
  before enabling `agent_job` result-ack scope.
