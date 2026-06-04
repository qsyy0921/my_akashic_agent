# TDD: QQ Group Media Live Smoke

## Scope

- Protect the Go OneBot delivery adapter change that tolerates non-echo
  WebSocket message frames while waiting for the echo response.
- Protect the repo-owned QQ group live smoke runbook surface.

## Target Code Paths

- `services/agent-runtime/infrastructure/onebotdelivery/adapter.go`
- `services/agent-runtime/infrastructure/onebotdelivery/websocket.go`
- `services/agent-runtime/infrastructure/onebotdelivery/adapter_test.go`
- `scripts/run-qq-group-live-smoke.ps1`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| websocket ignored event has array message | Go unit/integration | adapter ignores the frame and still returns the echoed action response |
| websocket group image dispatch | Go unit/integration | `send_group_msg` succeeds for image segments over WebSocket |
| group live smoke script | ATDD/manual | real group text/image/file sends are reported with explicit success or blocker evidence |

## Required Automated Tests

- `C:\Users\10495\AppData\Local\Programs\Go\bin\go.exe test ./infrastructure/onebotdelivery` from `services/agent-runtime`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

## Deferred Coverage

- Real NapCat group image/file upload semantics remain ATDD/live smoke because
  they depend on the operator's logged-in QQ session and the remote platform's
  rich-media behavior.
- The PowerShell script is validated through ATDD/live smoke rather than a mock
  harness.
