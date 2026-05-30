# Review: external lease gate

Spec:
- `docs/sdd/specs/agent-gateway/021-external-queue-backend.md`

Implementation summary:
- Added `external_lease` gate diagnostics to `/v1/queue-backend`.
- The gate reports explicit cutover state, required checks, blockers, and
  ack/nack/retry/dead-letter/rollback policies.
- `AKASHIC_QUEUE_MODE=external_lease` alone does not enable external lease
  execution. The current gate remains blocked because the executor is not
  implemented.
- This slice does not add a new service. It keeps the behavior inside
  `services/agent-runtime` and reuses the existing DDD/hexagonal boundary.

Tests run:
- `go test ./...` from `services/agent-runtime`.
- `go build ./cmd/agent-runtime` from `services/agent-runtime`.

Findings:
- No correctness blockers found.
- The gate intentionally remains blocked even when cutover and dual-read smoke
  flags are true, because `executor_implemented=false` is hard-coded for this
  slice.
- This keeps the migration observable without creating another partial runtime
  service or accidental external lease execution path.

Decision:
- Proceed with the gate. The next implementation should be the smallest useful
  executor inside `agent-runtime`, starting with outbox delivery only.

Follow-ups:
- Run local NATS JetStream smoke before setting `AKASHIC_QUEUE_DUAL_READ_SMOKE_PASSED=true`.
- Implement the first external lease executor by reusing existing outbox
  lifecycle and dispatch services rather than splitting another service.
