# Review: runtime overview outbound cutover plan

Spec: `docs/sdd/specs/agent-gateway/073-runtime-overview-outbound-cutover-plan.md`

Implementation summary:

- Added optional `OutboundCutoverPlan` dependency to `RuntimeOverviewService`.
- Runtime overview now calls the read-only plan with default smoke matrix and
  `desired_execution_owner=auto`.
- Summary now exposes plan readiness, decision, blocker count and execution
  owners.
- Added `Outbound Cutover` card with plan detail for dashboard/operator entry.

Tests run:

- `go test ./app/service -run TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics -count=1 -v`
- `go test ./cmd/agent-runtime -run TestRuntimeConfig -count=1 -v`
- `go test ./...`
- `go build ./cmd/agent-runtime`
- `git diff --check` (only LF/CRLF warnings)

Findings:

- The overview call remains read-only; the plan service does not mutate env,
  enqueue deliveries, start workers or dispatch adapters.
- Python remains responsible for AI reasoning and content creation.

Decision:

- Accepted. This improves operator visibility for QQ/NapCat Go delivery
  cutover without changing live sending behavior.

Follow-ups:

- Live cutover still requires manual QQ/NapCat smoke and explicit env changes.
