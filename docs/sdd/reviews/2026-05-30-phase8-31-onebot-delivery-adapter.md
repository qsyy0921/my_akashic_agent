# Review: OneBot/NapCat Delivery Adapter

Spec: `docs/sdd/specs/agent-gateway/018-onebot-napcat-delivery-adapter.md`

Implementation summary:
- Added `infrastructure/onebotdelivery` as a Go outbound adapter behind the
  existing `DeliveryAdapter` port.
- Added `conversation_type` to delivery dispatch steps so the adapter can choose
  private versus group OneBot methods without leaking transport details into the
  domain planner.
- Added env-gated OneBot endpoint wiring in `cmd/agent-runtime`.
- Left QR login, inbound WebSocket observation, and QQ direct fallback unchanged.

Tests run:
- `go test ./...` from `services/agent-runtime`.

Findings:
- DDD dependency direction is preserved. Domain and app layers know only delivery
  steps and adapter ports; OneBot HTTP details stay in infrastructure.
- QQ Go delivery is not enabled unless endpoint env vars are present, so existing
  observe-only group collection is not disrupted.
- Multi-account route safety depends on explicit channel aliases such as
  `qq_1049511700` and `qq_2365524513`.

Decision:
- Approved as an observe-safe migration slice. It is suitable to merge before
  live QQ cutover.

Follow-ups:
- Configure actual NapCat OneBot HTTP endpoints and run a live smoke for
  private text, group text, image, and file.
- After smoke passes, add selected QQ aliases to
  `integrations.agent_runtime.outbound_channels`.
- Add contract fixtures for outbox delivery and media content in the next
  migration slice.
