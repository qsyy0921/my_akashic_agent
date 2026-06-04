# TDD: QQ Live Send Smoke

## Scope

- Protect the Go/Python boundary and endpoint contract used by the local QQ live
  smoke runbook.

## Target Code Paths

- `services/agent-runtime/trigger/http/handler.go`
- `services/agent-runtime/app/service/message_send_service.go`
- `services/agent-runtime/app/service/delivery_dispatch_service.go`
- `services/agent-runtime/infrastructure/onebotdelivery/*`
- `integrations/agent_gateway.py`
- `scripts/run-qq-live-smoke.ps1`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| outbound enqueue contract | Go integration | `/v1/outbound` accepts a QQ outbox delivery payload |
| dispatch send contract | Go integration | `/v1/delivery-dispatch/send` returns `sent` results and provider ids |
| unavailable adapter fallback | Go integration | missing adapter returns `sender_unavailable` |
| Python client mapping | pytest | `dispatch_outbox_delivery()` maps success and dispatch errors correctly |
| live smoke script execution | ATDD/manual | real QQ private-text smoke remains runtime-only and is not faked by unit tests |

## Required Automated Tests

- `C:\Users\10495\AppData\Local\Programs\Go\bin\go.exe test ./...` from `services/agent-runtime`
- `uv run pytest tests/test_agent_gateway_client.py -q`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

## Deferred Coverage

- Real QQ private/group/image/file sends remain ATDD/live smoke because they
  depend on live NapCat sessions and real platform delivery semantics.
- The PowerShell script itself is exercised through the ATDD path rather than a
  synthetic unit-test harness.
