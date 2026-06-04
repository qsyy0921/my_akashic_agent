# Review: goal verifier observe-only silence check

Spec:
- `docs/sdd/specs/agent-gateway/158-goal-verifier-observe-only-silence.md`

Implementation summary:
- Extended `scripts/verify-go-migration-goal.ps1` so it now reads synced
  observe targets, probes one real observe-only QQ group route through
  delivery-dispatch readiness, and separately confirms that a QQ private route
  still plans successfully.
- Added stable `open_blockers` entries for missing observe targets, missing
  hard block enforcement, and private-route regression.

Tests run:
- `.\scripts\verify-go-migration-goal.ps1`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

Findings:
- Unified goal verification now reports observe-only silence evidence directly,
  instead of relying on manual turn summaries.
- The current local runtime still blocks real observe-only QQ group routes and
  still allows QQ private readiness.

Decision:
- Accept.

Follow-ups:
- Keep the observe-only silence check in the unified verifier as the default
  goal-update path.
