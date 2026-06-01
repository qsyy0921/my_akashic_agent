# SPEC-114: Dashboard Media Content Access Plan Proxy

## Status

Accepted for the current iteration.

## Context

Go runtime exposes `GET /v1/media-assets/content-access-plan?asset_id=...`.
The dashboard content route now uses it as an error fallback, but the frontend
cannot query the plan directly before opening the attachment. A direct dashboard
proxy lets the UI show access state, blockers, and suggested steps without first
streaming content or causing a failed image load.

## Boundary Analysis

Go owns:

- media asset content access policy;
- content root checks and readiness reasons;
- blockers, endpoint, and operational steps.

Python dashboard owns:

- a read-only HTTP proxy to Go runtime;
- error normalization for the browser API.

Out of scope:

- Python reimplementing media root policy;
- Python downloading remote media;
- content streaming from the plan endpoint;
- OCR/VLM/RAG/AI or file semantic parsing;
- metadata/file mutation.

## Decision

Add:

```text
GET /api/dashboard/media-assets/content-access-plan?asset_id=...
```

The endpoint calls Go:

```text
GET /v1/media-assets/content-access-plan?asset_id=...
```

and returns the Go `data` object directly. It must preserve `side_effect=none`
and return stable HTTP errors when the runtime is not configured or unavailable.

## Acceptance

- Successful proxy returns Go plan detail.
- Runtime HTTP errors preserve status code and message.
- Runtime network/JSON errors return dashboard-level 502.
- Existing media content proxy tests continue to pass.
- Target Python tests, Go regression, and diff check pass.
