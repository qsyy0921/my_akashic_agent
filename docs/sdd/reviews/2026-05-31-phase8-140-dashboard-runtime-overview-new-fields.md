# Phase 8.140 Review: Dashboard Runtime Overview New Fields

## Scope

Updated the Python runtime overview dashboard adapter to expose the newest
Go-owned `/v1/runtime-overview` diagnostics for delivery smoke readiness and
media asset content readiness.

## Changes

- Added summary defaults for `delivery_smoke_*` and `media_asset_content_*`
  fields.
- Normalized `delivery_smoke_readiness` with the existing dashboard helper.
- Added a read-only normalizer for `media_asset_content_diagnostics`.
- Returned both detail blocks from `_normalize_go_runtime_overview`.
- Extended the dashboard plugin test fixture to cover cards, summary defaults,
  delivery smoke detail, and media asset content detail.

## Boundary Check

Go remains the source of truth for deterministic runtime diagnostics:

- delivery smoke readiness;
- media asset content readiness;
- runtime overview summary/card/detail payloads.

Python dashboard only adapts the read model for the frontend. It does not send
QQ/Telegram messages, lease outbox or AgentJob work, mutate runtime state,
perform OCR/VLM/file parsing, or call AI models.

## Verification

- `uv run pytest --basetemp .\.tmp\pytest tests/test_runtime_overview_dashboard_plugin.py -q`

Result: `5 passed`.

## Risks

- The dashboard now exposes content readiness, not semantic understanding.
  Image OCR/VLM and file parsing still belong to Python AI workers.
- If the Go runtime overview is unavailable, legacy fallback cards remain
  intentionally narrower than the full Go aggregate.
