# Review: text-only outbox cutover gate

Spec:
- `docs/sdd/specs/agent-gateway/144-text-only-outbox-cutover-gate.md`

Implementation summary:
- Added a runtime-config guard to `AgentGatewayOutboxWorker` so the Python
  compatibility outbox worker backs off whenever Go local outbox ownership is
  active.
- Extended `scripts/start-agent-runtime.ps1` to propagate
  `AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS`, making repo-local text-only cutover
  reproducible without ad hoc shell edits.
- Re-ran live outbox ownership checks: Go local worker still auto-sends text,
  while image/file deliveries remain queued under `text_only`.

Tests run:
- `uv run pytest tests/test_agent_gateway_outbox_worker.py -q`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `POST /v1/outbound` text smoke followed by `/v1/outbox/{event_id}`
- `GET /v1/outbox-events?limit=20`
- `POST /v1/outbound` image/file gate smokes followed by repeated
  `/v1/outbox/{event_id}` polling
- `/v1/runtime-config`
- `/v1/queue-backend`
- `/v1/outbound-cutover/readiness`
- `/v1/agent-worker-statuses`
- `/v1/knowledge-job-planner/readiness`
- `/v1/jobs?type=group_memory_extract&limit=10`

Findings:
- Before the fix, rich-media deliveries were still leased by
  `akashic-python-worker:outbox`, so the Go `text_only` gate was incomplete.
- After the fix and process restart, Python outbox worker reports
  `reason=go_runtime_outbox_worker_active`, real text deliveries are leased by
  `agent-runtime-outbox-worker`, and real image/file deliveries stay
  `queued/attempts=0`.
- Telegram backend is still unconfigured on this machine:
  `telegram_token_configured=false` and no Telegram receiver is present.

Decision:
- Accept. Partial cutover is now real: QQ text can run on Go local outbox
  ownership while rich media remains safely gated until the NapCat / QQ
  platform blocker is removed.

Follow-ups:
- After NapCat / QQ rich-media session capability is restored, re-run group
  image/file smoke and decide whether to widen `outbox_allowed_kinds`.
- If desired, make the repo-local startup profile persistently opt into
  `text_only` rather than relying on operator-set environment values.
