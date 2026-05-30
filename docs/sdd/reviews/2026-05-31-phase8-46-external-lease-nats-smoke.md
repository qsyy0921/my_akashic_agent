# Review: external lease NATS smoke

Spec:
- `docs/sdd/specs/agent-gateway/021-external-queue-backend.md`

Implementation summary:
- Added a repeatable Go smoke test under `services/agent-runtime/smoke`.
- The smoke connects to a local NATS JetStream DSN only when
  `AKASHIC_NATS_SMOKE_DSN` is set; otherwise default `go test ./...` skips it.
- The smoke wires real `ExternalLeaseConsumer`, real `Publisher`, real
  `WorkQueueExternalLeaseService`, in-memory Go state stores, and a fake
  DeliveryAdapter.
- The fake adapter records dispatch steps and never calls QQ, Telegram, OneBot,
  or any platform network API.
- Added delayed NATS nack support for retryable external lease failures so a
  retryable platform error does not become an immediate redelivery loop.

Smoke result:
- success outbox delivery: Go `succeeded`, NATS `ack`;
- retryable dispatch failure: Go `failed -> queued`, delayed NATS `nack`;
- terminal failure after max attempts: Go `dead_lettered`, NATS `ack`;
- unsupported outbox subject payload: NATS `term`;
- fake adapter received only the three real outbox delivery steps.

Tests run:
- `go test ./...` from `services/agent-runtime`.
- `go build ./cmd/agent-runtime` from `services/agent-runtime`.
- Docker NATS smoke:
  `AKASHIC_NATS_SMOKE_DSN=nats://127.0.0.1:4222 go test ./smoke -run TestExternalLeaseNATSSmokeOutboxDispositions -count=1 -v`.

Findings:
- No correctness blocker found.
- The smoke validates the external queue boundary without real sends, but it
  still intentionally avoids enabling QQ/Telegram runtime cutover.
- Generic `agent_job` external lease remains out of scope.

Decision:
- Treat outbox `external_lease` ack/nack/term as smoke-tested locally.

Follow-ups:
- Before any real QQ/NapCat cutover, configure OneBot WebSocket endpoints,
  stop legacy state-store outbox workers, then run an explicit real-send smoke
  with user confirmation.
- Review Python worker idempotency before moving `agent_job` leasing to NATS.
