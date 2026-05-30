# Review: queue shadow publish diagnostics

Spec:
- `docs/sdd/specs/agent-gateway/021-external-queue-backend.md`

Implementation summary:
- Added a `WorkQueuePublishDiagnosticReader` application port and a
  `queuediagnostics.Recorder` infrastructure decorator around the NATS
  `shadow_publish` publisher.
- Extended `/v1/queue-backend` with `shadow_publish` diagnostics, including
  publish attempts, successes, failures, per-subject counts, and sampled
  reconciliation against Go outbox/job state stores and lifecycle event
  streams.
- Runtime still commits Go aggregate state before queue publish, and queue
  publish failures remain non-fatal after state commit.

Tests run:
- `go test ./...` from `services/agent-runtime`.

Findings:
- No correctness blockers found in this slice.
- Reconciliation is intentionally sampled at 200 items because the current
  repository list contracts are bounded query APIs, not full analytic scans.
- Diagnostics are in-memory for the active runtime process; durable queue audit
  can be added if shadow publish needs restart-spanning publish accounting.

Decision:
- Proceed with this slice. It improves observability without enabling external
  MQ leasing or changing Python worker behavior.

Follow-ups:
- Implement `dual_read_compare`: consume NATS work notifications, validate each
  candidate against Go state-store leases, and record mismatches without
  executing side effects.
- Run live NATS smoke only after a local NATS JetStream server is started.
