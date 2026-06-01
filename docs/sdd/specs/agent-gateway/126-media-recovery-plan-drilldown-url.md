# SPEC-126: Media Recovery Plan Drilldown URL

## Status

Accepted for the current iteration.

## Context

`SPEC-125` added a Go-owned read-only media content recovery plan, but operators
still have to know the endpoint by convention unless they are already inside the
plan page. Message detail and content diagnostics should expose a deterministic
link to the recovery plan in the same way they already expose content and
content access plan links.

## Boundary Analysis

Go owns:

- deterministic media asset URL hints in runtime read models;
- content diagnostics fields that point to Go-owned drilldown endpoints;
- stable route names for access/recovery plans.

Python owns:

- dashboard projection and browser-friendly URLs;
- OCR, VLM, file parsing, semantic extraction, RAG and AI execution.

Out of scope:

- downloading or restoring media content;
- invoking OCR/VLM/RAG/AI;
- frontend control actions or mutation buttons.

## Decision

Add a recovery-plan drilldown URL alongside existing access-plan links:

- Go `MediaAssetContentDiagnosticItemView` adds
  `content_recovery_plan_endpoint`.
- Python dashboard media asset enrichment adds
  `content_recovery_plan_url`.
- Dashboard TypeScript media asset type includes the optional field so UI code
  can render or inspect it without using raw JSON.

All fields are deterministic strings derived from `asset_id`. They do not call
the recovery-plan endpoint during list/detail enrichment and therefore stay
read-only.

## Acceptance

- Go service tests assert diagnostics items include the recovery-plan endpoint.
- Python dashboard tests assert list/detail enriched media assets include the
  browser recovery-plan URL.
- Existing recovery-plan endpoint tests remain green.
- TODO is cleared after SDD DONE/LIVE_CHECKS/OPEN_ISSUES/review/index updates.
