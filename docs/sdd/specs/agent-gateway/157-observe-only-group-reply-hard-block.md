# Spec 157: Observe-Only Group Reply Hard Block

## Why

`observe_only=true` and `reply_allowed=false` already exist in synced QQ observe
target config, but that was still a configuration contract. After partial Go
outbox cutover and richer per-route delivery gates, group silence needs a hard
runtime invariant: even if an outbound event is created for an observe-only QQ
group, delivery dispatch must refuse to plan a reply.

## Requirements

1. Go delivery dispatch planning must reject any outbound route that exactly
   matches an enabled observe target with `reply_allowed=false`.
2. The rejection must happen before adapter planning or platform send, and must
   return a stable route error with message
   `observe-only target does not allow replies`.
3. The hard block must apply to QQ group routes only when the route matches a
   synced observe target; private QQ routes and non-matching routes must still
   plan normally.
4. Repo-owned live verification must prove both sides:
   - a real observe-only QQ group route returns the route error above from
     `/v1/delivery-dispatch/readiness`
   - a private QQ route still returns `ready=true`

## Non-Goals

- Changing how observe targets are configured or synced from Python.
- Disabling all QQ group sends globally.
- Changing Telegram behavior or rich-media platform blockers.
