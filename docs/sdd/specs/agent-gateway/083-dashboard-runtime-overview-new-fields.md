# Dashboard Runtime Overview New Fields

## Context

Go runtime overview now exposes two newer control-plane details:

- `delivery_smoke_readiness`
- `media_asset_content_diagnostics`

The Python dashboard plugin already prefers `/v1/runtime-overview`, but its
normalization layer predates these fields. Without explicit normalization, the
dashboard cannot rely on stable top-level detail blocks or default summary keys
when the Go aggregate is partially unavailable or when tests exercise older
payloads.

## Boundary

Go owns the runtime control-plane state and diagnostics:

- delivery smoke readiness matrix and adapter routing checks;
- media asset content readiness;
- runtime overview summary/card/detail payloads.

Python dashboard owns only presentation/read adaptation:

- normalizing Go JSON into the dashboard contract;
- providing default zero values for missing summary keys;
- preserving old fallback behavior when Go runtime overview is unavailable.

Python dashboard must not:

- send QQ/Telegram messages;
- lease outbox or AgentJob work;
- mutate runtime state;
- perform OCR/VLM/file parsing;
- make AI or model calls.

## Design

Extend `plugins/runtime_overview/dashboard.py`:

1. Add summary defaults for:
   - `delivery_smoke_ready`
   - `delivery_smoke_reason`
   - `delivery_smoke_cases`
   - `delivery_smoke_ready_cases`
   - `delivery_smoke_not_ready_cases`
   - `delivery_smoke_blockers`
   - `media_asset_content_assets`
   - `media_asset_content_ready`
   - `media_asset_content_forbidden`
   - `media_asset_content_unavailable`
   - `media_asset_content_disabled`
   - `media_asset_content_error`
2. Normalize `delivery_smoke_readiness` with the existing
   `_normalize_delivery_smoke_readiness` helper.
3. Add a small read-only normalizer for `media_asset_content_diagnostics`.
4. Return both details from `_normalize_go_runtime_overview`.

The legacy fallback `_overview_cards` may stay as-is unless Go aggregate is
unavailable; the new fields are primarily Go-owned aggregate payloads.

## Verification

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q`

## Risks

- Frontend components that only render cards are already covered by Go cards;
  this change mainly stabilizes detail access.
- The media asset content detail still only means deterministic content access,
  not successful AI understanding.
