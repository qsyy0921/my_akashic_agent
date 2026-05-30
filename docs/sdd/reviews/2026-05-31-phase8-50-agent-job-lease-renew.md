# Review: agent job lease renew

Spec:
`docs/sdd/specs/agent-gateway/021-external-queue-backend.md`

Implementation summary:
- Added `AgentJob.RenewLease` and `RenewAgentJobLeaseCommand`.
- Added `POST /v1/jobs/{job_id}/renew` to extend the active lease without
  incrementing attempts.
- Added `AgentJobEventRenewed` so renewals are visible in the durable job event
  stream.
- Added `AgentGatewayClient.renew_job` and a shared Python
  `AgentJobLeaseHeartbeat` helper.
- Image, knowledge, and RAG-eval Python workers now start a background lease
  renew task during job execution and stop it before result/failure writeback.

Tests run:
- `go test ./...`
- `uv run pytest tests\test_agent_gateway_heartbeat.py tests\test_agent_gateway_client.py tests\test_agent_gateway_image_worker.py tests\test_agent_gateway_knowledge_worker.py tests\test_agent_gateway_rag_eval_worker.py tests\test_agent_gateway_knowledge_worker_smoke.py -q --basetemp .tmp\pytest-agent-renew`

Findings:
- Renew requires a non-empty current lease token and rejects stale tokens.
- Renew preserves the job status and attempt count, so it is safe for long AI
  work without changing retry semantics.
- The Python heartbeat helper no-ops when no token is present, preserving legacy
  compatibility.

Decision:
Treat lease renew as the second result-ack building block. Keep generic
`agent_job` on state-store leasing until strict token mode, queue-id exact
leasing, and NATS ack/nack contract tests are implemented.

Follow-ups:
- Add strict token cutover once old workers are retired.
- Design NATS delivery ack after Python result writeback, not at Go lease time.
