# Phase 8 Review 176: Go Contract Test Media Access Plan

## Scope

- Add Go contract fixture coverage for `media_asset_content_access_plan.qq.image.json`.
- Assert the Go-owned shape of `MediaAssetContentAccessPlan`.
- Keep the slice test-only; no runtime, dashboard, media streaming, OCR/VLM/RAG, or AI behavior changes.

## Design Check

- Go now requires the access plan fixture alongside existing runtime boundary fixtures.
- The test validates stable operator-facing fields: `ready`, `reason`, `side_effect`, runtime/dashboard paths, content endpoint, content URL, and key required/verify/fallback steps.
- Python remains responsible for dashboard behavior and its own fixture checks; Go only validates the shared contract from the runtime boundary.

## Verification

- `go test ./...` from `services/agent-runtime`
- `uv run pytest tests/test_sdd_contract_fixtures.py -q`
- `git diff --check`

## Risk

- The test intentionally checks stable contract fields instead of every optional blocker/detail field, so access plan internals can evolve without forcing noisy fixture churn.

## Result

Accepted. Go and Python now both validate the media content access plan contract fixture.
