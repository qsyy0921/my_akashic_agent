# Review: NATS live smoke

Spec:
- `docs/sdd/specs/agent-gateway/021-external-queue-backend.md`

Implementation summary:
- Ran a local NATS JetStream smoke with `AKASHIC_QUEUE_BACKEND=nats_jetstream`
  and `AKASHIC_QUEUE_MODE=dual_read_compare`.
- Started `agent-runtime` on `127.0.0.1:8784` with
  `AKASHIC_QUEUE_CONSUMER_CONCURRENCY=2` and
  `AKASHIC_QUEUE_MAX_IN_FLIGHT=8`.
- Created a local outbox work item through `/v1/outbound`; this did not invoke
  delivery dispatch and did not send any QQ or Telegram message.
- Verified `/v1/queue-backend` reported one successful shadow publish and one
  matched dual-read comparison.

Smoke result:
- provider: `nats_jetstream`
- mode / phase: `dual_read_compare`
- external queue active: `true`
- shadow attempts / successes / failures: `1 / 1 / 0`
- dual-read compared / matched / mismatched: `1 / 1 / 0`
- configured consumer concurrency: `2`
- configured max in-flight: `8`

Tests run:
- Local Docker NATS JetStream container: `nats:2-alpine -js`.
- `go build -o .tmp/nats-smoke/bin/agent-runtime-smoke.exe ./cmd/agent-runtime`
  from `services/agent-runtime`.
- HTTP smoke against `/healthz`, `/v1/outbound`, `/v1/outbox`, and
  `/v1/queue-backend`.

Findings:
- No queue correctness blocker found.
- The first failed smoke attempt was caused by the script checking
  `health.status` instead of the runtime envelope field `health.data.status`.
- A second script issue built from the repository root instead of the Go module
  root. The corrected smoke builds from `services/agent-runtime`.
- `GOPROXY=https://goproxy.cn,direct` and `GOSUMDB=off` are already suitable for
  local Go dependency fallback. External pulls should still use the local
  `7897` proxy when needed.

Decision:
- Treat NATS `shadow_publish` and `dual_read_compare` as locally smoke-tested.
- Do not enable `external_lease` execution yet; the gate must remain blocked
  until an executor exists and its cutover checks pass.

Follow-ups:
- Implement the smallest useful `external_lease` executor in
  `services/agent-runtime`, starting with outbox delivery only.
- Keep the state store authoritative and require Go lifecycle transitions
  before any ack/nack policy can execute real side effects.
