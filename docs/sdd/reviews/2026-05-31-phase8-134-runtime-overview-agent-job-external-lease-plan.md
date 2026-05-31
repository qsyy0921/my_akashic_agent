# Review: runtime overview agent_job external lease plan

Spec: `docs/sdd/specs/agent-gateway/077-runtime-overview-agent-job-external-lease-plan.md`

Implementation summary:

- Added optional `AgentJobExternalLeasePlan` dependency to `RuntimeOverviewService`.
- Runtime overview now calls the read-only plan path with bounded `limit`,
  `event_limit` and `stale_after_seconds` propagated into the embedded
  readiness check.
- `/v1/runtime-overview` now exposes `agent_job_external_lease_plan_*` summary
  fields, an `Agent Job External Lease Plan` card and full plan detail.
- `cmd/agent-runtime` wires the existing plan service into the overview
  aggregate.

Tests run:

- `go test ./app/service -run TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics -count=1 -v`
- `go test ./cmd/agent-runtime -run TestRuntimeConfig -count=1 -v`
- `go test ./...`
- `go build ./cmd/agent-runtime`

Findings:

- No mutation path was introduced. The aggregate only reads the plan and never
  leases AgentJob records, acknowledges queue messages, starts Python workers or
  changes environment variables.
- The card reports `warn` when the plan is blocked and `ok` when it is ready or
  ready to enable, matching the existing outbound cutover plan behavior.

Decision:

- Accept. Dashboard/operator overview now exposes both result-ack readiness and
  the cutover plan from the Go runtime control plane.

Follow-ups:

- Live-smoke `/v1/runtime-overview` against a running runtime and compare it
  with `/v1/agent-job-external-lease/plan` before enabling real NATS
  result-ack.
