# ATDD: Observe-Only Group Reply Hard Block

## Scenario

The local Go runtime runs with synced QQ observe targets, and at least one
enabled observe-only group target has `reply_allowed=false`.

## Acceptance Checks

1. `GET /v1/observe-targets` shows enabled QQ group observe targets with
   `observe_only=true` and `reply_allowed=false`.
2. Creating a real outbox event for one of those exact group routes and calling
   `POST /v1/delivery-dispatch/readiness` returns:
   - `HTTP 400`
   - `message="observe-only target does not allow replies"`
   - `data.error_kind="route_error"`
3. Creating a private QQ outbox event and calling
   `POST /v1/delivery-dispatch/readiness` still returns `ready=true`.

## Failure Signals

- Observe-only group readiness returns `ready=true`.
- The route is blocked only by config drift and not by runtime dispatch.
- Private QQ readiness is incorrectly blocked by observe-only rules.
