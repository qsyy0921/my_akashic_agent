# Review: Runtime Overview Delivery Smoke

Spec:

- `docs/sdd/specs/agent-gateway/014-runtime-dashboard-overview.md`
- `docs/sdd/specs/agent-gateway/018-onebot-napcat-delivery-adapter.md`

Implementation summary:

- Added `AgentGatewayClient.check_delivery_smoke_readiness()` for Python-side
  contract coverage of Go `POST /v1/delivery-smoke/readiness`.
- Added dashboard proxy
  `/api/dashboard/runtime-overview/delivery-smoke-readiness`, which posts to
  the Go endpoint and normalizes ready totals, blockers, cases, plans, status,
  and `side_effect=none`.
- Added a manual `Smoke Readiness` action to the `Delivery Adapters` detail
  panel, with optional comma-separated group ids. Normal overview loading does
  not call the smoke endpoint.

Tests run:

- `uv run pytest tests\test_agent_gateway_client.py tests\test_runtime_overview_dashboard_plugin.py -q --basetemp .tmp\pytest-delivery-smoke-dashboard`
- `npm run typecheck`
- Restarted local Python dashboard and called
  `/api/dashboard/runtime-overview/delivery-smoke-readiness?group_ids=27234224&include_synthetic_media=true`;
  the proxy returned 12 ready cases and `side_effect=none`.

Findings:

- The dashboard proxy uses POST only toward the Go smoke endpoint and does not
  create outbox records or call delivery send endpoints.
- The frontend action is manual, matching the safety posture used by adapter
  health checks.

Decision:

- Accept as operator visibility for the read-only QQ/NapCat delivery smoke
  gate.

Follow-ups:

- After operator confirmation, run live QQ/NapCat send smoke separately and
  keep recent-send / bot-protocol loop protection enabled.
