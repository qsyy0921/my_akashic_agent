# Spec 158: Goal Verifier Observe-Only Silence Check

## Why

`verify-go-migration-goal.ps1` is the repo-owned end-of-turn verifier for this
workspace. After observe-only QQ groups gained a hard runtime reply block, that
invariant should be checked by the same unified verifier instead of being
remembered as a one-off manual note.

## Requirements

1. `scripts/verify-go-migration-goal.ps1` must read current
   `GET /v1/observe-targets` state and report enabled QQ group targets with
   `reply_allowed=false`.
2. The unified verifier must create a live observe-only QQ group outbox event
   and confirm `POST /v1/delivery-dispatch/readiness` returns the stable block:
   - `message="observe-only target does not allow replies"`
   - `error_kind="route_error"`
3. The unified verifier must also confirm a QQ private route still returns
   `ready=true`, so the observe-only hard block does not regress private
   planning.
4. If any of the checks above fail, the unified verifier must add a stable
   blocker to `checks.open_blockers`.

## Non-Goals

- Adding a new global “all QQ groups never send” policy.
- Replacing dedicated live smoke scripts for rich-media or outbox scope.
- Changing Telegram or knowledge planner behavior.
