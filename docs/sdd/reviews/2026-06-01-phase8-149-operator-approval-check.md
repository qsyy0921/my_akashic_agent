# Review: Operator Approval Check

Spec: `docs/sdd/specs/agent-gateway/092-operator-approval-check.md`

Implementation summary:

- Added `OperatorApprovalCheck` query input and `OperatorApprovalCheckView` output.
- Extended the existing Go operator approval service with `CheckOperatorApproval`.
- Added `GET /v1/operator-approvals/check`.
- The check supports exact `approval_id` lookup or `target_kind + target_id` lookup.
- The endpoint returns stable `approved`, `reason`, `blockers`, optional `approval` and `side_effect=none`.

Tests run:

- `go test ./app/service -run TestOperatorApprovalService -count=1 -v`
- `go test ./trigger/http -run TestOperatorApproval -count=1 -v`
- Full regression commands are recorded in the iteration final response.

Findings:

- Target lookup uses the newest matching ledger record. This avoids accidentally allowing an older approved record after a newer rejected or revoked record for the same target.
- Exact approval id lookup can additionally validate target kind/id when clients provide them, preventing accidental use of a valid approval for the wrong plan.
- The API is still read-only and does not grant authorization by itself; it only provides a deterministic preflight answer for future control-plane mutation code.

Decision:

- Accept. This fills the missing control-plane seam between an audit ledger and future mutations without introducing mutation side effects.

Follow-ups:

- Future mutation endpoints must require an approval id and call this check before changing config, cutover state, worker concurrency or queue ownership.
- Real authorization, actor identity and replay protection still need a separate design before exposing mutation APIs beyond local/operator use.
