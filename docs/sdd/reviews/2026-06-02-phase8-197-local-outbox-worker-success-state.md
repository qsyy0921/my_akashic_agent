# Review: local outbox worker success state

Spec:

- `docs/sdd/specs/agent-gateway/140-local-outbox-worker-success-state.md`

Implementation summary:

- Removed the redundant `MarkDispatching` call from the local outbox worker.
- Updated the worker tests to reflect the lease-driven state transition.
- Verified the worker in a real local bring-up with
  `AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED=true`, then rolled the runtime back to
  the default worker-disabled state.

Tests run:

- `C:\Users\10495\AppData\Local\Programs\Go\bin\go.exe test ./trigger/job ./infrastructure/onebotdelivery`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- Live runtime verification with worker enabled:
  - `GET /v1/runtime-workers`
  - `POST /v1/outbound-cutover/readiness`
  - `POST /v1/outbound`
  - `GET /v1/outbox/{event_id}`
  - `GET /v1/send-ledger/recent?...`
- Live rollback verification with default local bring-up:
  - `GET /v1/runtime-config`
  - `POST /v1/outbound-cutover/readiness`

Findings:

- The worker bug was deterministic: the lease path already applied the
  `dispatching` transition, so the extra `MarkDispatching` call incorrectly
  exhausted single-attempt smoke deliveries.
- After the fix, a real private-text outbox item was sent automatically by the
  worker and reached `status=succeeded`.
- The rich-media blocker remains separate: QQ group image/file still fail at the
  current NapCat/QQ session boundary.

Decision:

- Accepted.

Follow-ups:

- Keep default local bring-up on `AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED=false`
  until the remaining QQ rich-media blocker is resolved.
- Re-run local worker smoke after rich-media remediation and before any durable
  cutover decision.
