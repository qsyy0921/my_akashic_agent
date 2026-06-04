# ATDD: Local Runtime Account Channel Mapping

## Scenario

Repo-local Go runtime is restarted through `scripts/start-agent-runtime.ps1`
with the local outbox worker enabled and account/kind gate settings preserved.

## Acceptance Checks

1. `GET /v1/runtime-workers` shows the outbox worker running and exposes:
   - `1049511700=qq_1049511700`
   - `2365524513=qq_2365524513`
2. `.\scripts\verify-go-outbox-scope-live.ps1` returns:
   - `first_account_private_file.condition_met=true`
   - `second_account_group_file.condition_met=true`
   - `first_account_group_file_gated.condition_met=true`
   - `first_account_private_image_gated.condition_met=true`
3. `.\scripts\verify-go-migration-goal.ps1 -IncludeOutboxScopeSmoke ...` no
   longer includes `outbox_scope_case_failed_second_account_group_file` in
   `checks.open_blockers`.

## Failure Signals

- `second_account_group_file` dead-letters while native NapCat `file_send`
  still succeeds.
- Outbox worker attributes omit per-account channel mapping.
- Unified goal verifier reports a false outbox scope blocker after restart.
