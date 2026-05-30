# Review: Delivery Smoke Readiness

Spec:

- `docs/sdd/specs/agent-gateway/015-delivery-dispatch-plan.md`
- `docs/sdd/specs/agent-gateway/018-onebot-napcat-delivery-adapter.md`

Implementation summary:

- Added `POST /v1/delivery-smoke/readiness` as a read-only preflight for
  QQ/NapCat and Telegram delivery routing.
- The app service builds synthetic in-memory deliveries, reuses the domain
  `DeliveryPlanner`, and checks only `DeliveryAdapter.SupportsDeliveryChannel`.
- Runtime defaults generate dual-account QQ private text/image/file cases from
  `AKASHIC_BOT_IDS`, optional group cases from
  `AKASHIC_DELIVERY_SMOKE_GROUP_IDS`, and account-channel mappings from
  configured OneBot aliases or `AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT`.

Tests run:

- `go test ./...` under `services/agent-runtime`
- Rebuilt and restarted local `agent-runtime`, then called
  `POST /v1/delivery-smoke/readiness` with group `27234224`; 12 read-only
  private/group text/image/file cases returned ready with `side_effect=none`.
- Rechecked `GET /v1/delivery-adapters/health?timeout_seconds=5`; `qq`,
  `qq_1049511700`, `qq_2365524513`, and `telegram` were healthy and
  authenticated.

Findings:

- No platform send API is called by the readiness endpoint; unit and HTTP tests
  assert that adapter dispatch steps remain empty.
- Synthetic image/file cases verify planner and adapter routing only. They do
  not prove NapCat media upload behavior, so the live send smoke remains a
  separate cutover gate.

Decision:

- Accept this as the read-only gate before QQ/NapCat live send smoke.

Follow-ups:

- Run live send smoke only after operator confirmation: dual QQ private text,
  group text, image, and file.
- If live smoke passes, choose either explicit
  `integrations.agent_runtime.outbound_channels` cutover or the Go local outbox
  worker cutover, then re-check recent-send and bot-protocol loop protection.
