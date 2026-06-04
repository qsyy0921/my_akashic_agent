# TDD: MQ adapter boundary dashboard read model

## Scope

- Protect dashboard normalization of `queue_backend` capability-matrix fields.

## Target Code Paths

- `E:\agent\my-akashic_agent\plugins\runtime_overview\dashboard.py`
- `E:\agent\my-akashic_agent\tests\test_runtime_overview_dashboard_plugin.py`
- `E:\agent\my-akashic_agent\scripts\verify-mq-adapter-boundary.ps1`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| Go runtime overview includes queue backend capability matrix | integration | dashboard reader preserves `supported_providers` / `provider_capabilities` / `selected_provider_capability` |
| MQ boundary verifier reads runtime + dashboard | live verification | selected provider, NATS recommendation, and planned-only Redis/Rabbit classification stay visible |
| Unified goal verifier includes MQ boundary classification | live verification | `mq_adapter_boundary` appears in current-turn residual classification |

## Required Automated Tests

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

## Deferred Coverage

- Real Redis Streams / RabbitMQ adapter implementation remains out of scope;
  this slice only protects the read-model and live classification boundary.
