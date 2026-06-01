# SPEC-133: Media Content Diagnostics Recovery Preflight Endpoint

## Status

Accepted for the current iteration.

## Context

Go owns media content diagnostics, content access plan, content recovery plan,
and now approval-bound content recovery preflight. Dashboard message rows can
already synthesize a `content_recovery_preflight_url`, but Go diagnostics still
only expose content/access/recovery-plan drilldowns. Operators inspecting
runtime overview should be able to discover the recovery preflight from the same
Go-owned diagnostics payload.

## Boundary Analysis

Go owns:

- media asset content diagnostics item shape;
- deterministic endpoint hints for content, access plan, recovery plan, and
  recovery preflight;
- runtime overview data consumed by dashboard plugins.

Dashboard owns:

- read-only display of links already present in runtime overview detail.

Out of scope:

- calling the preflight during list/detail rendering;
- creating approvals or mutation audits;
- executing media download/restore/cache operations;
- OCR/VLM/RAG/AI work.

## Decision

Add `content_recovery_preflight_endpoint` to
`MediaAssetContentDiagnosticItemView`, populated as:

```text
/v1/media-assets/content-recovery/preflight?asset_id=<urlencoded asset_id>
```

The runtime overview dashboard `Media Asset Content` table adds a `preflight`
link when present. Rendering remains purely local to `card.detail`.

## Acceptance

- Go media content diagnostics include `content_recovery_preflight_endpoint`.
- HTTP/media tests assert the field is present.
- Runtime overview dashboard plugin asset includes and renders the preflight
  link.
- Contract fixture covers the recovery preflight URL.
- SDD DONE/LIVE_CHECKS/OPEN_ISSUES/review/index are updated.
- TODO is cleared after verification.
