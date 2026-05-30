# Review: agent job lease work

Spec:
`docs/sdd/specs/agent-gateway/021-external-queue-backend.md`

Implementation summary:
- Added `AgentJobLeaseWorkCommand` and `AgentJobService.LeaseWork`.
- Added `POST /v1/jobs/lease-work` as the queue-notification lease entrypoint
  for `agent_job` work.
- The endpoint leases the exact `work_id`, validates `work_kind=agent_job`, and
  rejects mismatched `aggregate_id`.
- Added `AgentGatewayClient.lease_work` for Python-side contract tests and
  future queue-driven workers.
- Updated external queue diagnostics wording so the remaining blockers are now
  strict-token deployment, timeout recovery, and ack-after-result duplicate
  delivery handling.

Tests run:
- `go test ./...`
- `uv run pytest tests\test_agent_gateway_client.py -q --basetemp .tmp\pytest-lease-work`

Findings:
- NATS work notifications already publish `work_id=job.JobID`, so exact work
  leasing can reuse the existing aggregate lease path without adding a new
  domain model.
- This endpoint does not execute Python work and does not acknowledge NATS; it
  is only the exact leasing primitive required before a queue result-ack worker
  can exist.

Decision:
Treat `lease-work` as the queue contract boundary for future `agent_job`
external lease. Keep `lease-next` for current state-store worker polling.

Follow-ups:
- Add timeout recovery and duplicate-delivery contract smoke before any NATS
  ack/nack mapping for generic jobs.
