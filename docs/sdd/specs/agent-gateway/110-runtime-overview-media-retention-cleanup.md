# SPEC-110: Runtime Overview Media Retention Cleanup

## Status

Accepted for the current iteration.

## Context

The Go runtime now owns media retention diagnostics, cleanup plan, preflight,
and metadata-only cleanup execution. Operators still need one read-only runtime
overview card that answers:

- are there expired media metadata candidates;
- is the cleanup executor visible;
- did recent cleanup audits apply or fail.

Without this aggregate, dashboards would need to duplicate retention plan and
control mutation filtering logic.

## Boundary Analysis

Go owns:

- retention cleanup readiness visibility;
- cleanup endpoint discoverability;
- recent control mutation audit summary for `media_asset_retention`;
- stable runtime overview fields for frontends.

Python owns:

- OCR/VLM/file parsing;
- semantic memory/RAG extraction from attachments;
- AI decisions and provider-specific behavior.

Out of scope:

- executing cleanup from runtime overview;
- creating approvals or mutation audits;
- deleting media metadata or files;
- calling Python, MQ, RAG, OCR/VLM, or AI.

## Decision

Add `media_asset_retention_cleanup` to runtime overview. It is derived from:

- `MediaAssetRetentionPlan`;
- `ControlMutationAudits` filtered to `target_kind=media_asset_retention`.

The view exposes:

- candidate and asset counts;
- readiness reason and blockers from the plan;
- endpoints for plan, preflight, dry-run/apply;
- recent cleanup audits and status totals;
- `side_effect=none`.

## Acceptance

- Runtime overview service test covers summary/card/detail.
- HTTP aggregate test keeps returning the new detail shape.
- Full Go regression passes.
- TODO is cleared at the end of the iteration.
