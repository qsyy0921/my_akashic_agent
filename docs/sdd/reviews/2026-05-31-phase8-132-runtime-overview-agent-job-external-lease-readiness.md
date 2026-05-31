# Review: runtime overview agent_job external lease readiness

Spec: `docs/sdd/specs/agent-gateway/075-runtime-overview-agent-job-external-lease-readiness.md`

Implementation summary:

- Added an optional `AgentJobExternalLeaseReady` dependency to `RuntimeOverviewService`.
- Runtime overview now calls the read-only `CheckAgentJobExternalLeaseReadiness` path with the same bounded filter values used by other overview diagnostics.
- `/v1/runtime-overview` now exposes `agent_job_external_lease_*` summary fields, an `Agent Job External Lease` card, and full readiness detail.
- `cmd/agent-runtime` wires the existing `AgentJobExternalLeaseReadinessService` into the overview aggregate.

Tests run:

- `go test ./app/service -run TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics -count=1 -v`
- `go test ./cmd/agent-runtime -run TestRuntimeConfig -count=1 -v`
- `go test ./...`
- `go build ./cmd/agent-runtime`

Findings:

- No new mutation path was introduced. The aggregate only reads Go runtime diagnostics and Python worker heartbeat status.
- The card uses `danger` when Python worker coverage is not ready, because result-ack cutover without a healthy Python AI worker would strand AI jobs.

Decision:

- Accept. The dashboard/operator entrypoint now sees the same readiness gate as `/v1/agent-job-external-lease/readiness`.

Follow-ups:

- Live-smoke `/v1/runtime-overview` against a running runtime and compare it with `/v1/agent-job-external-lease/readiness` before enabling real NATS result-ack cutover.
