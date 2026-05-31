# Phase 8.105 Review: Runtime Overview External Lease Diagnostics

Spec:

- `docs/sdd/specs/agent-gateway/048-runtime-overview-external-lease-diagnostics.md`

Implementation summary:

- Runtime overview summary now includes external lease execution totals,
  error totals, and ack/nack/term disposition counters.
- Runtime overview now has an `external_lease_diagnostics` card with status
  derived from diagnostics: `danger` on executor errors, `warn` on nack/term,
  `ok` on successful executions, and `muted` when diagnostics are absent.
- The card keeps the original queue backend detail so dashboard users can still
  inspect raw external lease gate and diagnostic samples.

Tests run:

- `go test ./app/service -run TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics`
- `go test ./...`

Findings:

- No blocking findings. The change is read-only aggregation over the existing
  queue backend view and does not add side effects.
- Live NATS smoke still needs to confirm summary counters match
  `/v1/queue-backend.external_lease.diagnostics` under a real consumer.

Decision:

- Accepted.

Follow-ups:

- Use the runtime overview card during NATS external lease smoke before enabling
  broader queue cutover.
