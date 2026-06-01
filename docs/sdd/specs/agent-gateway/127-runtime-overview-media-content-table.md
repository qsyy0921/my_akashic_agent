# SPEC-127: Runtime Overview Media Content Table

## Status

Accepted for the current iteration.

## Context

`media_asset_content` is already a Go-owned runtime overview detail and includes
stable content, access-plan, and recovery-plan endpoints. The dashboard plugin
currently renders every card detail as raw JSON, which keeps the data available
but makes attachment diagnostics hard to operate from the UI.

This is tracked by OI-008: some dashboard drilldowns still require manual JSON
reading. The next safe step is presentation-only: render the existing
Go-provided media content diagnostics as a compact read-only table.

## Boundary Analysis

Go owns:

- media asset registry, content diagnostics, content access plan and recovery
  plan endpoints;
- runtime overview summary/card/detail data.

Python/dashboard owns:

- read-only projection and browser-friendly display of Go-owned data.

Out of scope:

- fetching content bytes from the table renderer;
- calling access/recovery plan endpoints during render;
- downloading/restoring media;
- OCR, VLM, RAG, file parsing or AI execution;
- mutation buttons or control-plane actions.

## Decision

When the runtime overview card id is `media_asset_content`, the dashboard panel
renders:

- totals for assets / ready / forbidden / unavailable / disabled / error;
- a bounded table of diagnostic items with asset id, kind/name, status, reason;
- links to `content_endpoint`, `content_access_plan_endpoint`, and
  `content_recovery_plan_endpoint` when present;
- the raw JSON detail below the table as fallback/debug evidence.

The renderer only consumes the already loaded `card.detail`. It must not make
additional network requests.

## Acceptance

- Plugin JS asset contains the media-content table renderer and recovery-plan
  link labels.
- Existing runtime overview dashboard plugin tests remain green.
- SDD DONE/LIVE_CHECKS/OPEN_ISSUES/review/index are updated.
- TODO is cleared after verification.
