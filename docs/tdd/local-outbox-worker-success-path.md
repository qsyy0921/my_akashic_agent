# TDD: Local Outbox Worker Success Path

## Scope

- Protect the local outbox worker state transition so a leased delivery is not
  re-dispatched into `dead_lettered` before `MarkSucceeded`.

## Target Code Paths

- `services/agent-runtime/trigger/job/outbox_delivery_worker.go`
- `services/agent-runtime/trigger/job/outbox_delivery_worker_test.go`
- `services/agent-runtime/domain/model/outbox_delivery.go`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| leased delivery dispatch success | Go unit | worker calls `LeaseNext`, dispatches, then marks `succeeded` without a second dispatch-state mutation |
| dispatch error | Go unit | worker records `MarkFailed` with the surfaced delivery error kind |
| no delivery | Go unit | worker exits idle without dispatch or outbox mutation |
| rate-limited account | Go unit | worker skips blocked accounts without dispatching or mutating a second time |

## Required Automated Tests

- `C:\Users\10495\AppData\Local\Programs\Go\bin\go.exe test ./trigger/job` from `services/agent-runtime`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

## Deferred Coverage

- Real worker bring-up and automatic QQ send remain ATDD/live smoke because they
  depend on the live NapCat session and real platform delivery.
