# Review: Dashboard Media Recovery Preflight Proxy

## Scope

- Proxy the Go-owned media content recovery preflight through the Python
  dashboard.
- Expose deterministic preflight drilldown URLs on message media assets.

## Result

- `GET /api/dashboard/media-assets/content-recovery/preflight` forwards
  `asset_id`, `target_id`, `operator_id`, and `approval_id` to Go
  `/v1/media-assets/content-recovery/preflight`.
- Message list/detail media assets now include `content_recovery_preflight_url`.
- List/detail enrichment only adds URLs; it does not call the preflight
  endpoint.

## Boundary

Go remains the source of truth for recovery plan/preflight semantics, approval
checks and control mutation preflight. Python dashboard is a read-only
projection/proxy. It does not create approvals or mutations, download, restore,
cache, stream, parse, or invoke OCR/VLM/RAG/AI.

## Tests

- `uv run pytest tests/test_dashboard_api.py -q`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `git diff --check`
