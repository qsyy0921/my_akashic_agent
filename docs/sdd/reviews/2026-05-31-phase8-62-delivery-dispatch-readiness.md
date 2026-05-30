# Review: Phase 8.62 Delivery Dispatch Readiness

## Scope

- Added Go-owned `POST /v1/delivery-dispatch/readiness`.
- The endpoint reuses the Go delivery planner and configured DeliveryAdapters
  to report whether an outbox delivery can be dispatched by Go.
- Added `AgentGatewayClient.check_outbox_dispatch_readiness()` for Python
  tooling and future cutover logic.

## Design

- Readiness lives in the existing `DeliveryDispatchService`; no new service was
  introduced.
- The use case returns the dispatch plan, `ready`, `reason`, and
  `missing_channels`.
- It deliberately does not call `DispatchDeliveryStep`, so it cannot send QQ,
  Telegram, files, images, or text.
- Existing Python outbox worker behavior is unchanged. This slice only moves
  the deterministic adapter-readiness decision behind Go for inspection and
  later controlled cutover.

## Verification

- `go test ./app/service ./trigger/http -run "TestDeliveryDispatch" -count=1 -v`
- `uv run pytest tests\test_agent_gateway_client.py -q --basetemp .tmp\pytest-dispatch-readiness-client`

## Remaining Risk

- This does not prove a live NapCat or Telegram endpoint is healthy. It proves
  that Go can determine configured adapter support for a planned outbox route
  without side effects. Live send smoke remains explicit and operator-approved.
