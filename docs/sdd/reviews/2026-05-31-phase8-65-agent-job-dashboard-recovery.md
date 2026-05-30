# Review: Phase 8.65 Agent Job Dashboard Recovery

Spec:
- `docs/sdd/specs/agent-gateway/008-agent-job-orchestration.md`
- `docs/sdd/specs/agent-gateway/021-external-queue-backend.md`

Implementation summary:
- Added dashboard API `POST /api/dashboard/agent-jobs/recover-expired`.
- The API delegates to Go `POST /v1/jobs/recover-expired` and normalizes the
  scanned, recovered, dead-lettered, and item results.
- Agent Jobs detail view now has a manual expired-lease recovery action.
- Job normalization exposes `lease_token_present` instead of the raw
  `lease_token` value.

Tests run:
- `uv run pytest tests\test_agent_jobs_dashboard_plugin.py -q --basetemp .tmp\pytest-agent-jobs-recovery-dashboard`
- `uv run pytest tests\test_agent_jobs_dashboard_plugin.py tests\test_agent_gateway_client.py -q --basetemp .tmp\pytest-agent-jobs-recovery-regression`
- `npm run typecheck`
- `npm run build:plugins`
- `git diff --check`

Findings:
- The action is deterministic control-plane recovery only. It does not execute
  Python model workers, send platform messages, or acknowledge NATS messages.
- Keeping token visibility as a boolean preserves useful diagnostics without
  leaking fencing tokens into the browser UI.

Decision:
Accept as a safe Go-owned lifecycle operations slice.

Follow-ups:
- Add throughput/dead-letter trend metrics for `agent_job`.
- Keep live `agent_job` NATS result-ack cutover behind the existing strict
  token and smoke-test gates.
