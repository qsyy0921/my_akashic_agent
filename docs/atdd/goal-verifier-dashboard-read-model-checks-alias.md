# ATDD: Goal Verifier Dashboard Read-Model Checks Alias

## Scenario

As an operator reading the canonical goal-verifier artifact, I need dashboard
read-model checks to be available from both the legacy top-level path and the
`checks.*` path so machine readers do not misclassify verified evidence as
missing.

## Acceptance

1. Run `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`.
2. Open `.codex-goal-verifier.json`.
3. Confirm both of these paths exist and match:
   - `dashboard_read_models.proactive_tick_logs_readable`
   - `checks.dashboard_read_models.proactive_tick_logs_readable`
4. Confirm no live blocker classification changes:
   - `goal_ready_to_close=false`
   - blockers remain external-state driven unless runtime truth changed.
