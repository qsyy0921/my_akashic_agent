# Review: outbound cutover plan

Spec: `docs/sdd/specs/agent-gateway/072-outbound-cutover-plan.md`

Implementation summary:

- Added `POST /v1/outbound-cutover/plan`.
- The endpoint accepts the delivery smoke matrix plus optional
  `desired_execution_owner`.
- The plan embeds outbound readiness and returns required checks, enable steps,
  verification endpoints, rollback steps, blockers and `side_effect=none`.
- Go owns deterministic cutover sequencing; Python remains responsible for AI
  reasoning, model/tool execution and compatibility send paths before explicit
  cutover.

Tests run:

- `go test ./app/service -run TestOutboundCutoverPlan -count=1 -v`
- `go test ./trigger/http -run TestOutboundCutoverPlanEndpointReturnsReadOnlyPlan -count=1 -v`
- `go test ./cmd/agent-runtime -run "TestRuntimeConfig|TestDeliverySmokeReadinessConfig|TestQueueBackend" -count=1 -v`
- `go test ./...`
- `go build ./cmd/agent-runtime`
- `git diff --check` (only LF/CRLF warnings)

Findings:

- The endpoint is read-only and does not call delivery adapters, enqueue
  outbox records, mutate jobs or write environment variables.
- Local worker and NATS external lease plans deliberately include rollback
  steps because both paths can create real platform side effects after an
  operator enables them.

Decision:

- Accepted for the next runtime control-plane slice.

Follow-ups:

- Run real QQ/NapCat live send smoke before setting cutover env flags.
