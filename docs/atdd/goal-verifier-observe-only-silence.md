# ATDD: Goal Verifier Observe-Only Silence Check

## Scenario

The repo-owned unified goal verifier runs against the local runtime after
observe-only QQ groups have been synced into Go.

## Acceptance Checks

1. `.\scripts\verify-go-migration-goal.ps1` returns an
   `observe_only_silence` section.
2. That section reports:
   - at least one enabled QQ group target with `reply_allowed=false`
   - `checks.observe_only_group_route_blocked=true`
   - `checks.private_route_still_ready=true`
3. `checks.open_blockers` does not contain:
   - `observe_only_targets_missing`
   - `observe_only_group_reply_block_not_enforced`
   - `observe_only_private_route_regressed`

## Failure Signals

- The unified verifier omits observe-only silence evidence.
- A real observe-only QQ group route is not blocked.
- The private QQ route becomes unplannable after adding the observe-only check.
