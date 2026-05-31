# Review: Outbound Cutover Readiness

Spec:

- `docs/sdd/specs/agent-gateway/071-outbound-cutover-readiness.md`

Implementation summary:

- Added `OutboundCutoverReadinessService` in the Go app layer.
- Added `POST /v1/outbound-cutover/readiness`.
- The endpoint aggregates sanitized runtime config, delivery smoke readiness,
  queue backend execution owner and runtime worker diagnostics.
- Readiness blocks on incomplete OneBot config, failed smoke matrix, or no
  running Go outbox execution path.
- Main wiring registers the endpoint with existing runtime services; Python code
  is unchanged.

Tests run:

- `go test ./app/service -run TestOutboundCutoverReadiness -count=1 -v`
- `go test ./trigger/http -run TestOutboundCutoverReadinessEndpointReturnsReadOnlyGate -count=1 -v`
- `go test ./cmd/agent-runtime -run "TestDeliverySmokeReadinessConfig|TestRuntimeConfig|TestQueueBackend" -count=1 -v`
- `go test ./...`
- `go build ./cmd/agent-runtime`
- `git diff --check`

Findings:

- Adapter availability alone is not sufficient for cutover; the endpoint also
  verifies the outbox execution owner and worker path.
- The endpoint reuses delivery smoke planning and does not dispatch adapter
  steps.
- Python remains responsible for AI-generated message creation and compatibility
  callers until explicit cutover.
- Full regression exposed an existing time-dependent knowledge pipeline test;
  the test now pins the agent-worker status service clock to the scenario time.

Decision:

- Accepted pending full regression. This is a safe read-only gate before
  enabling Go execution for QQ/NapCat sends.

Follow-ups:

- Run live send smoke for both QQ accounts and selected groups before enabling
  Go local outbox worker or NATS external lease in production.
