# Review: Runtime Overview Control Audit

Spec: `docs/sdd/specs/agent-gateway/094-runtime-overview-control-audit.md`

Implementation summary:

- Added operator approval and control mutation audit dependencies to `RuntimeOverviewService`.
- Added runtime overview summary fields for approval totals, active approvals, decision counts, mutation totals and mutation status counts.
- Added `Control Audit` runtime overview card with bounded detail containing `operator_approvals` and `control_mutations`.
- Wired the existing Go services into `cmd/agent-runtime/main.go`.
- Kept the aggregate read-only: it only calls list APIs and never records approvals, records mutations, checks approvals, mutates config, starts workers, touches MQ or executes Python AI.

Tests run:

- `go test ./app/service -run TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics -count=1 -v`
- `go test ./cmd/agent-runtime -run TestRuntimeConfig -count=1 -v`
- Full regression commands are recorded in the iteration final response.

Findings:

- `Control Audit` card is `warn` when there are no approvals or when failed/rolled_back mutation records exist. That keeps operator attention on missing approval discipline or failed control-plane attempts.
- Runtime overview intentionally does not call approval check preflight because that would imply a target-specific decision. The overview only summarizes ledger state.

Decision:

- Accept. This makes Go-owned control-plane audit status visible from the same operator surface as queue, worker, outbox, knowledge and receiver diagnostics.

Follow-ups:

- Python dashboard can consume the new `control_audit` card and detail if it needs a dedicated table.
- Future real mutation endpoints still need explicit SDD for approval-id binding, mutation audit-id binding, rollback execution and authorization.
