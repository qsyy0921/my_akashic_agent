# Review: Media Content Recovery Preflight

## Scope

- Add a Go-owned read-only media content recovery preflight endpoint.
- Add `media_asset_content/recover_content` to the control mutation policy
  allowlist.
- Keep the future download/restore/cache executor out of scope.

## Result

- `GET /v1/media-assets/content-recovery/preflight` reads the existing content
  recovery plan and blocks when recovery is not needed.
- For actionable recovery candidates, it runs control mutation preflight with
  `target_kind=media_asset_content` and `action=recover_content`.
- An active approval returns `ready=true` and a suggested planned audit; missing
  approval returns stable blockers.

## Boundary

Go owns deterministic media metadata, content readiness, recovery planning,
operator approval checks and control mutation preflight. Python continues to own
OCR/VLM/file parsing, semantic interpretation and AI-driven recovery
experiments. This slice does not download, restore, cache, stream, parse, create
approval records, or create mutation audit records.

## Tests

- `go test ./...`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `git diff --check`
