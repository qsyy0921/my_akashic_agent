# SPEC-112: Media Asset Content Access Plan

## Status

Accepted for the current iteration.

## Context

The runtime already stores QQ/Telegram media asset metadata and exposes
`/v1/media-assets/{asset_id}/content`. When content access fails, the frontend or
dashboard currently sees a raw HTTP error such as forbidden or unavailable. The
batch `/v1/media-assets/content-diagnostics` endpoint can explain content state,
but it is optimized for aggregate diagnostics rather than a single user click.

## Boundary Analysis

Go owns:

- deterministic media asset existence checks;
- content reader availability and local path policy checks;
- stable content endpoint and blocker reporting;
- side-effect-free runtime API for the frontend.

Python owns:

- OCR, VLM, file parsing, semantic extraction, and AI summaries;
- provider-specific fallbacks and prompt/tool orchestration.

Out of scope:

- opening or streaming content in the plan endpoint;
- downloading remote media;
- changing content access policy;
- OCR/VLM/RAG/AI calls;
- deleting metadata or files.

## Decision

Add a Go-owned read-only content access plan endpoint:

```text
GET /v1/media-assets/content-access-plan?asset_id=...
```

The endpoint returns:

- `ready`, `reason`, and `blockers`;
- the normalized `asset` metadata;
- `content_endpoint`;
- optional content mime/size when a safe deterministic probe succeeds;
- required, verify, and fallback steps;
- `side_effect=none`.

The plan may briefly open and immediately close local content through the
existing content reader, matching content diagnostics behavior. It must not
stream the bytes to the caller or trigger AI processing.

## Acceptance

- Missing `asset_id`, missing asset, disabled reader, forbidden path,
  unavailable file, and ready content are represented with stable reasons.
- HTTP route is read-only and rejects non-GET methods.
- Go tests cover service and handler behavior.
- `go test ./...` passes.
- TODO is cleared at the end of the iteration.
