# TDD: Dashboard runtime plan drilldowns

## Scope

- Protect runtime overview panel rendering for plan-style control-plane detail.

## Target Code Paths

- `E:\agent\my-akashic_agent\plugins\runtime_overview\dashboard_panel.ts`
- `E:\agent\my-akashic_agent\tests\test_runtime_overview_dashboard_plugin.py`
- `E:\agent\my-akashic_agent\scripts\verify-go-migration-goal.ps1`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| Panel asset exposes capacity/priority labels | asset/integration | JS contains `Agent Job Capacity`, `Agent Job Priority` |
| Panel asset exposes knowledge/outbound cutover labels | asset/integration | JS contains `Knowledge Planner Cutover`, `Outbound Cutover Plan`, `Preview Bucket`, `Expected OneBot Channels` |
| Unified verifier reflects new dashboard read-model evidence | live verification | `dashboard_read_models` marks all four tables true and `dashboard_fallback` upgrades to runtime read-model live verification |

## Required Automated Tests

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

## Deferred Coverage

- Real cutover / worker execution remains out of scope; this slice only improves
  dashboard readability and goal-verifier evidence.
