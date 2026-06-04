# TDD: dashboard media recovery runtime-overview read model

## Scope

- Protect dashboard normalization of Go runtime-overview
  `media_asset_content_recovery` summary/card/detail.

## Target Code Paths

- `E:\agent\my-akashic_agent\plugins\runtime_overview\dashboard.py`
- `E:\agent\my-akashic_agent\tests\test_runtime_overview_dashboard_plugin.py`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| Go runtime overview includes media recovery detail but no card | integration | dashboard reader synthesizes `media_asset_content_recovery` card and top-level detail |
| Dashboard reader falls back because Go aggregate is unavailable | integration | payload still contains `media_asset_content_recovery` top-level field and muted/unknown card |
| Live runtime overview is slower than generic short timeout | live verification | dashboard reader still uses Go aggregate instead of silently downgrading to fallback |

## Required Automated Tests

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

## Deferred Coverage

- Real HTTP/HTTPS recovery execution remains in ATDD/live verifier, because this
  slice only changes dashboard read-model behavior.
