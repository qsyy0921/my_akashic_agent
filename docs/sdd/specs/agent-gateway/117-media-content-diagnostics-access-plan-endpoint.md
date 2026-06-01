# SPEC-117: Media Content Diagnostics Access Plan Endpoint

## Status

Accepted for the current iteration.

## Context

Go `content-access-plan` explains a single media asset's content readiness and
blockers. Go `content-diagnostics` exposes batch readiness, but each item only
contains `content_endpoint`. Dashboard/runtime consumers must reconstruct the
single-asset plan endpoint themselves.

## Boundary Analysis

Go owns:

- media asset content diagnostics;
- runtime API path for content access plans;
- deterministic access status and side-effect-free plan lookup.

Python owns:

- runtime overview dashboard normalization and presentation.

Out of scope:

- fetching access-plan detail for every diagnostics item;
- streaming content;
- OCR/VLM/RAG/AI or file semantic parsing;
- remote media download or metadata/file mutation.

## Decision

Add `content_access_plan_endpoint` to each
`MediaAssetContentDiagnosticItemView`, populated as:

```text
/v1/media-assets/content-access-plan?asset_id=<escaped asset id>
```

Python dashboard normalization should preserve this field and fall back to the
same deterministic endpoint when older Go runtimes do not return it.

## Acceptance

- Go service/HTTP tests cover the new field.
- Python runtime overview dashboard test validates normalized detail.
- `go test ./...` and target Python test pass.
- TODO is cleared at the end of the iteration.
