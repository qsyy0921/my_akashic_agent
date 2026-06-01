# SPEC-118: Agent Runtime Media API Docs

## Status

Accepted for the current iteration.

## Context

Recent slices added Go-owned media asset content diagnostics, content access
plans, retention diagnostics, cleanup preflight, metadata cleanup execution,
runtime overview aggregation, dashboard proxies, and contract fixtures. The Go
runtime README still documents only the older media registry/content endpoints,
and the SDD spec index stops at the early media content diagnostics entries.

## Boundary Analysis

Go owns:

- media asset metadata registry;
- content diagnostics and content access plan;
- retention diagnostics, cleanup plan, preflight, and metadata cleanup executor;
- runtime API contract and side-effect declarations.

Python owns:

- dashboard proxy/presentation;
- OCR, VLM, file parsing, RAG, and semantic extraction.

Out of scope:

- changing runtime behavior;
- executing cleanup;
- adding frontend UI logic;
- OCR/VLM/RAG/AI or media download.

## Decision

Update:

- `services/agent-runtime/README.md` media API section;
- `docs/sdd/specs/agent-gateway/000-index.md` media-related entries.

The docs must explicitly state:

- content access plan is read-only and does not stream content;
- content diagnostics items link to access plans;
- retention cleanup preflight is side-effect-free;
- retention cleanup executor deletes Go metadata only and requires approval;
- local file deletion and AI processing remain out of scope.

## Acceptance

- README lists current media endpoints.
- SDD index references specs 089 through 117 where applicable.
- `go test ./...` still passes.
- TODO is cleared at the end of the iteration.
