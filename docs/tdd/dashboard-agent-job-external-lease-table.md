# TDD: Dashboard agent_job external lease table

## Scope

- Protect runtime overview panel rendering for:
  - `agent_job_external_lease_readiness`
  - `agent_job_external_lease_plan`

## Target Code Paths

- `E:\agent\my-akashic_agent\plugins\runtime_overview/dashboard_panel.ts`
- `E:\agent\my-akashic_agent\tests\test_runtime_overview_dashboard_plugin.py`
- `E:\agent\my-akashic_agent\scripts\verify-go-migration-goal.ps1`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| Panel asset exposes structured external-lease labels | asset/integration | JS contains `Agent Job External Lease`, `Current Owner`, `Required Checks`, `Rollback Steps` |
| Runtime overview still serves normalized external-lease detail | integration | dashboard payload keeps `agent_job_external_lease_readiness` / `agent_job_external_lease_plan` |
| Unified verifier reflects dashboard read-model evidence | live verification | `dashboard_fallback` no longer relies only on proactive dashboard evidence |

## Required Automated Tests

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

## Deferred Coverage

- Real `agent_job` external lease result-ack cutover remains out of scope;
  this slice only improves dashboard readability and verifier evidence.
