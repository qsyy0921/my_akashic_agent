# Review: agent job external lease temp NATS smoke verifier

Spec:
- `docs/sdd/specs/agent-gateway/199-agent-job-external-lease-temp-nats-smoke-verifier.md`

Implementation summary:
- Added `scripts/verify_agent_job_external_lease_nats_smoke.py` and
  `scripts/verify-agent-job-external-lease-nats-smoke.ps1` to pull a temporary
  `nats:2-alpine` image, start an isolated JetStream container on a random
  localhost port, run the repo-owned Go smoke tests, and emit structured JSON.
- Updated `scripts/verify-go-migration-goal.ps1` to include the smoke output as
  `agent_job_external_lease_nats_smoke` and reuse its conclusion for
  `residual_classification.agent_job_external_lease_result_ack_smoke`.
- Replaced fixed historical timestamps in
  `services/agent-runtime/smoke/external_lease_nats_smoke_test.go` with current
  UTC timestamps so the flow smoke no longer expires its own lease on later
  calendar dates.

Tests run:
- `go test ./smoke -run "TestExternalLeaseNATSSmoke(AgentJobDuplicateTerminalAck|AgentJobPendingRunningSucceededFlow)" -count=1 -v`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-agent-job-external-lease-nats-smoke.ps1`
- `uv run pytest tests/test_verify_agent_job_external_lease_nats_smoke.py -q`

Findings:
- The temp-NATS smoke now passes on the current machine for both duplicate
  terminal ack and pending/running/succeeded flow.
- The previous failure was a test-time drift bug, not a NATS/provider/runtime
  protocol failure.
- Production runtime cutover is still blocked by current config and owner
  gates; the new smoke only proves the repo-owned Go path is ready to verify.

Decision:
- Accept.

Follow-ups:
- When NATS cutover intent is explicit, combine this temp-NATS smoke evidence
  with the existing live runtime verifier before enabling result-ack flags.
