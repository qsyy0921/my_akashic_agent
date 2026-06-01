# Phase 8 Review 179: Media Content Access Plan URL Fields

## Scope

- Extend Go `MediaAssetContentAccessPlanView` with `runtime_path`, `dashboard_path`, and `content_url`.
- Generate deterministic URL hints from `asset_id` in Go.
- Update Go service and HTTP tests plus Python dashboard/contract regression tests.

## Design Check

- Go owns media asset route construction and access policy, so URL hints belong in the Go runtime response.
- Python dashboard remains a read-only proxy/presentation layer and does not duplicate content access policy.
- The change does not stream content, download remote media, run OCR/VLM/RAG, or invoke AI.

## Verification

- `go test ./app/service ./trigger/http` from `services/agent-runtime`
- `uv run pytest tests/test_sdd_contract_fixtures.py tests/test_dashboard_api.py -q`
- `go test ./...` from `services/agent-runtime`
- `uv run pytest tests/test_sdd_spec_index.py -q`
- `git diff --check`

## Risk

- Dashboard URL paths are now part of the runtime response contract. Future dashboard route changes must update Go access plan tests and shared contract fixtures together.

## Result

Accepted. Media content access plans now expose deterministic runtime and dashboard URL fields directly from Go.
