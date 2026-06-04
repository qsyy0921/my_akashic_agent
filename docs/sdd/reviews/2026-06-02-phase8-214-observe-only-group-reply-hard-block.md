# Review: observe-only group reply hard block

Spec:
- `docs/sdd/specs/agent-gateway/157-observe-only-group-reply-hard-block.md`

Implementation summary:
- Delivery dispatch planning now consults synced observe targets before adapter
  planning.
- Exact QQ group routes that match enabled observe targets with
  `reply_allowed=false` now fail fast with a stable route error.
- Added unit tests for blocked observe-only group routes and still-allowed
  private routes.
- Re-ran live readiness against one real observe-only group and one private QQ
  route.

Tests run:
- `C:\Users\10495\AppData\Local\Programs\Go\bin\go.exe test ./app/service ./cmd/agent-runtime`
- Live `GET /v1/observe-targets`
- Live `POST /v1/delivery-dispatch/readiness` for `qq:1049511700:group:27234224`
- Live `POST /v1/delivery-dispatch/readiness` for `qq:1049511700:private:2365524513`

Findings:
- All synced observe-only QQ groups currently report `reply_allowed=false`.
- Real readiness for `qq:1049511700:group:27234224` now returns
  `HTTP 400 / observe-only target does not allow replies / route_error`.
- Real readiness for a QQ private route still returns `ready=true`.

Decision:
- Accept.

Follow-ups:
- Keep using this hard block even if later outbox gate scope expands.
- If the requirement changes from “observe-only groups never reply” to “all QQ
  groups never reply”, handle that as a broader policy slice.
