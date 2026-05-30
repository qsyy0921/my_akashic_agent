# Review: delivery dispatch plan

Spec:
- `docs/sdd/specs/agent-gateway/015-delivery-dispatch-plan.md`

Implementation summary:
- Added Go domain `DeliveryPlanner` to convert an outbox delivery into
  deterministic send steps.
- Added app/port/query/assembler layers for `DeliveryDispatchPlan`.
- Added `POST /v1/delivery-dispatch/plan`.
- Python `AgentGatewayClient` can request a plan and preserve Go error kinds.
- Python compatibility outbox worker now prefers the Go plan, then calls
  `message_push` only for the actual platform send.
- Legacy Python planning remains as fallback if an older runtime lacks the plan
  endpoint.

Tests run:
- `go test ./domain/service ./trigger/http`
- `uv run pytest tests/test_agent_gateway_client.py tests/test_agent_gateway_outbox_worker.py -q --basetemp .tmp/pytest-delivery-dispatch-plan`

Findings:
- Platform SDK calls should not move in the same slice as planning. Keeping
  send execution in Python preserves QQ/Telegram behavior while moving stable
  routing and payload-shaping rules to Go.
- The compatibility fallback is still needed during rolling restarts or when a
  user runs an older `agent-runtime` binary.

Decision:
- Accept as the first DeliveryAdapter migration slice. Go now owns dispatch
  planning; Python remains a thin sender adapter.

Follow-ups:
- Move Telegram HTTP and OneBot/NapCat send adapters behind Go
  `DeliveryAdapter` after plan telemetry is stable.
- Add cross-language contract fixtures for outbox delivery and dispatch plan.
