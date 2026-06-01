# Phase 8 Review 193: Dashboard Media Content Recovery Table

## Scope

- Added a specialized runtime overview dashboard renderer for
  `media_asset_content_recovery`.
- Built plugin assets.
- Updated focused dashboard plugin test assertions and SDD ledgers.

## Boundary Review

- The dashboard reads only the card detail already returned by Go runtime
  overview.
- It renders links to plan/preflight/recovery endpoints but does not call them.
- It does not create approvals, create mutation audits, download/cache content,
  update media registry, or trigger Python AI jobs.

## Verification

- `npm run build:plugins`
- `npm run typecheck`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q` with
  `TMP/TEMP/TMPDIR` pointed at `.tmp/pytest-tmp` because the default Windows
  temp pytest root was access-denied in this session.

## Residual Risk

- Other runtime overview cards still use raw JSON fallback and should be
  tabled only when there is clear operator value.
