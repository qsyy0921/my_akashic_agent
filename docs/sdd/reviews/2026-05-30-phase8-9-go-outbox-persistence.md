# Review: Go outbox persistence

Spec:
- `docs/sdd/specs/agent-gateway/006-outbox-delivery-retry.md`
- Continue moving stable delivery infrastructure to Go while keeping Python
  platform adapters as the compatibility sender until cutover is reviewed.

Implementation summary:
- Added `services/agent-runtime/infrastructure/outboxstore`, a file-backed
  implementation of `OutboxRepository` and `OutboxQueue`.
- Added `AKASHIC_OUTBOX_DSN` and `AKASHIC_OUTBOX_PATH` runtime configuration.
- Wired `cmd/agent-runtime` so `/v1/outbound` and `/v1/outbox/*` can use the
  persistent store while message event fanout remains in-memory.
- Updated the outbox SDD spec and runtime README with the new recovery contract.

Tests run:
- `gofmt -w services/agent-runtime/cmd/agent-runtime/main.go services/agent-runtime/infrastructure/outboxstore/store.go services/agent-runtime/infrastructure/outboxstore/store_test.go`
- `go test ./...` under `services/agent-runtime`
- `go build -o ..\..\.tmp\bin\agent-runtime-outbox-smoke.exe .\cmd\agent-runtime`
- Live runtime smoke on `127.0.0.1:8781` with `AKASHIC_OUTBOX_DSN`:
  - create `/v1/outbound`
  - mark `/v1/outbox/{event_id}/dispatching`
  - mark `/v1/outbox/{event_id}/failed`
  - restart runtime
  - verify `/v1/outbox/{event_id}` still reports `failed`

Findings:
- This is still not platform delivery cutover. QQ and Telegram SDK sends remain
  in Python until outbound adapters and shadow delivery are reviewed.
- Persisting outbox state first gives failed sends, retries, and eventual
  dead-letter records a stable control plane before Go owns actual dispatch.

Decision:
- Approved as a DDD infrastructure slice. It moves outbox delivery state into a
  recoverable Go-owned store without changing user-visible send behavior.

Follow-ups:
- Add dashboard outbox/error diagnostics.
- Add Go platform delivery adapters behind an outbound port.
- Add a lease/dequeue API when a separate delivery worker is introduced.
