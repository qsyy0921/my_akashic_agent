# Review: OneBot WebSocket Delivery

Spec: `docs/sdd/specs/agent-gateway/018-onebot-napcat-delivery-adapter.md`

Implementation summary:
- Extended the Go OneBot delivery adapter to support WebSocket action requests
  in addition to HTTP action endpoints.
- Added `AKASHIC_ONEBOT_WS_URLS`, `AKASHIC_ONEBOT_WEBSOCKET_URLS`,
  `AKASHIC_ONEBOT_WS_URL`, and `AKASHIC_ONEBOT_WEBSOCKET_URL` env parsing.
- Kept QQ runtime dispatch opt-in by wiring Python outbox worker runtime
  dispatch to `integrations.agent_runtime.outbound_channels`.
- Verified the live NapCat WebSocket endpoints with read-only
  `get_login_info` probes: `3001 -> 1049511700`, `3002 -> 2365524513`.

Tests run:
- `go test ./...` from `services/agent-runtime`.
- Read-only WebSocket `get_login_info` probe against local NapCat ports 3001 and
  3002.

Findings:
- Current NapCat configs have empty `httpServers` and enabled
  `websocketServers`. HTTP probes returning `426 Upgrade Required` are expected.
- WebSocket action dispatch is the least disruptive next step because it uses
  the already exposed ports and does not require container recreation.
- Live send smoke is intentionally still pending because it creates user-visible
  QQ messages.

Decision:
- Approved as a safer cutover foundation than requiring immediate NapCat HTTP
  endpoint changes.

Follow-ups:
- Set `AKASHIC_ONEBOT_WS_URLS` and tokens before restarting `agent-runtime`.
- Add selected QQ aliases to `integrations.agent_runtime.outbound_channels` only
  after an explicit live send smoke.
- Keep inbound observe-only WebSocket handling in Python until the inbound
  migration has its own design review.
