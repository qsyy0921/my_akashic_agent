# Review: Runtime Overview Media Content Table

## Scope

- Render the existing Go-owned `media_asset_content` runtime overview detail as
  a read-only table in the dashboard plugin.
- Preserve raw JSON fallback/debug output.

## Result

- The runtime overview panel now detects card id `media_asset_content` and
  renders totals plus a bounded media asset diagnostics table.
- Each row shows asset, name/kind, content status, reason, and deterministic
  links for content, access plan and recovery plan when present.
- The renderer uses only the already loaded `card.detail`; it does not call
  content, access-plan or recovery-plan endpoints during detail rendering.

## Boundary

Go remains the source of truth for media diagnostics and endpoint strings.
The dashboard only presents the data. It does not download or restore media,
open content streams, run OCR/VLM/file parsing, upload to RAG, or invoke AI.

## Tests

- `npm run build:plugins`
- `go test ./...`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `git diff --check`
