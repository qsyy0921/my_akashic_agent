# Review: Phase 8.71 Go Local Outbox Worker

Spec:
- `docs/sdd/specs/agent-gateway/006-outbox-delivery-retry.md`

Implementation summary:
- Added `trigger/job.OutboxDeliveryWorker`, an opt-in Go state-store worker for
  outbox delivery dispatch.
- The worker leases the next delivery through the outbox app service, marks it
  `dispatching`, calls the Go delivery dispatch app service, and then marks the
  delivery `succeeded` or `failed`.
- Added env-driven runtime startup behind
  `AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED=true`.
- Added a guard that rejects local worker startup when NATS external lease
  execution already owns outbox delivery.
- Documented the worker env configuration in README, config example, TODO, and
  the outbox SDD spec.

Tests run:
- `go test ./trigger/job`
- `go test ./cmd/agent-runtime`
- `go test ./...`
- `uv run pytest tests\test_agent_gateway_outbox_worker.py tests\test_bootstrap_wiring_p2.py -q --basetemp .tmp\pytest-go-local-outbox-worker-python-regression`

Findings:
- This moves another deterministic infrastructure loop from Python toward Go
  without changing default runtime behavior.
- The worker remains opt-in, so it does not trigger real QQ/Telegram sends
  unless the operator explicitly enables it and has configured DeliveryAdapters.
- The worker reuses existing app/domain transitions, so DDD boundaries remain:
  trigger/job schedules, app services lease/dispatch/update, domain owns
  lifecycle rules.

Decision:
Accept as the local state-store delivery executor before full external queue
cutover.

Follow-ups:
- Run a real Telegram or fake-adapter smoke before using it for QQ/NapCat.
- After QQ/NapCat live send smoke passes, decide whether local Go worker or
  NATS external lease should be the primary dispatch executor.
