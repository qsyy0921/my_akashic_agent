# Review: dashboard media asset metadata fallback

Spec:

- `docs/sdd/specs/agent-gateway/007-media-file-registry.md`

Implementation summary:

- Dashboard media proxy still uses Go `/v1/media-assets/{asset_id}/content` as the primary byte source.
- When Go returns `403` or `404`, the proxy now queries Go asset metadata and only falls back to a same-named file under the current workspace `uploads` directory.
- The fallback covers registered QQ media assets whose local mirror exists but whose Go safe-root configuration is stale or not yet loaded by the current runtime.

Tests run:

- `uv run pytest tests/test_dashboard_api.py -q --basetemp .tmp/pytest-dashboard-media-fallback`
- `uv run pytest tests/test_dashboard_api.py tests/test_bootstrap_wiring_p2.py tests/test_agent_gateway_outbox_worker.py tests/test_agent_gateway_client.py tests/test_support_modules.py -q --basetemp .tmp/pytest-media-and-runtime-wiring`
- `go test ./...` from `services/agent-runtime`

Findings:

- No test failures.
- Live dashboard and Go runtime were already returning `200 image/jpeg` for the reported QQ image asset after runtime safe-root recovery; this fallback makes the path robust for stale-root and old-link cases.

Decision:

- Accept. The compatibility path stays constrained to workspace `uploads` and does not broaden arbitrary local path access.

Follow-ups:

- Add Go/Python contract fixtures for media asset content and dashboard proxy behavior.
