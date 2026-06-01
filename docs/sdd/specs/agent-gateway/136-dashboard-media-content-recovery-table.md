# SPEC-136: Dashboard Media Content Recovery Table

## Status

Accepted for the current iteration.

## Context

`SPEC-135` added a Go-owned `media_asset_content_recovery` runtime overview
detail. The browser dashboard can show the raw JSON, but operators need a
scannable read-only table for applied/failed recovery audits and the relevant
plan/preflight/recovery links.

## Boundary Analysis

Go owns:

- runtime overview summary/card/detail payload;
- control mutation audit data;
- recovery plan, preflight, and executor endpoints.

Python/dashboard owns:

- rendering the already provided detail payload;
- linking to existing Go endpoints without invoking them.

Out of scope:

- calling recovery/preflight during render;
- creating approvals or mutation audits;
- executing recovery;
- starting Python OCR/VLM/RAG jobs;
- adding dashboard-side policy decisions.

## Decision

Add a specialized runtime overview dashboard renderer for card id
`media_asset_content_recovery`.

The renderer shows:

- KPI row for `audits`, `planned`, `applied`, `failed`, `rolled_back`;
- endpoint links for `plan`, `preflight`, and `recovery`;
- recent audit table with mutation id, target/action, status, approval,
  operator, reason, and timestamp;
- raw JSON fallback remains below the table.

## Acceptance

- Dashboard plugin JS contains `media_asset_content_recovery` specialized
  renderer wiring.
- The renderer uses existing detail fields only and does not call APIs.
- `npm run build:plugins`, `npm run typecheck`, and focused pytest pass.
- SDD DONE/LIVE_CHECKS/OPEN_ISSUES/PROJECT_STATUS/review/index are updated.
- TODO is cleared after verification.
