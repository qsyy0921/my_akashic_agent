# SPEC-116: Media Content Access Plan Contract

## Status

Accepted for the current iteration.

## Context

Go runtime now owns media content access planning, and Python dashboard exposes
read-only proxy links. Existing contract fixtures cover raw media content access
but not the new access-plan shape. Without a fixture, future Go/Python changes
could drift on field names such as `ready`, `reason`, `blockers`,
`content_endpoint`, or dashboard proxy paths.

## Boundary Analysis

Go owns:

- content access plan readiness, reason, blockers, endpoint, and side effect;
- deterministic media content access policy.

Python owns:

- dashboard proxy path naming;
- presenting the Go-owned plan without reinterpreting policy.

Out of scope:

- executing content access;
- OCR/VLM/RAG/AI or semantic parsing;
- remote media download;
- mutation of media metadata or files.

## Decision

Add a `MediaAssetContentAccessPlan` contract fixture under
`tests/fixtures/contracts` and make `tests/test_sdd_contract_fixtures.py`
validate it. The fixture documents both:

- Go runtime path: `/v1/media-assets/content-access-plan?...`
- Dashboard path: `/api/dashboard/media-assets/content-access-plan?...`

The contract requires `side_effect=none` and stable read-only plan fields.

## Acceptance

- Contract manifest includes the new fixture.
- Contract boundary test validates the access-plan shape.
- Python contract tests pass.
- Go regression still passes.
- TODO is cleared at the end of the iteration.
