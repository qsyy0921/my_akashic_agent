# Review: NATS Shadow Publish

Spec:

- `docs/sdd/specs/agent-gateway/021-external-queue-backend.md`

Implementation summary:

- Added provider-neutral `WorkQueuePublisher` out port for outbox deliveries
  and generic agent jobs.
- Added NATS JetStream adapter that creates/uses an `AKASHIC_WORK` stream and
  publishes work notifications under `akashic.work.*`.
- Added service integration after aggregate state and lifecycle events are
  committed.
- No external queue leasing or execution cutover in this slice.

Tests run:

- `go test ./...` from `services/agent-runtime`.

Findings:

- Shadow publish must be best effort after state commit, otherwise a transient
  MQ issue could make HTTP callers see failure while the aggregate already
  exists.
- Notification payloads must remain hints; workers still need Go lease APIs for
  authoritative state.
- NATS subject naming should avoid raw unsafe ids and use platform/account/job
  route partitions.

Decision:

- Proceed with NATS JetStream `shadow_publish`.
- Keep `external_lease` as a later, separately reviewed migration.

Follow-ups:

- Add queue publish diagnostics and reconciliation counts.
- Implement pull consumers with bounded goroutine worker pools after
  shadow-publish has been observed safely.
