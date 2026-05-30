# Review: dashboard worker contract smoke

Spec:

- `docs/sdd/specs/agent-architecture/006-contract-fixtures.md`
- `docs/sdd/specs/agent-gateway/010-knowledge-checkpoints.md`
- `docs/sdd/specs/agent-gateway/012-agent-job-event-stream.md`

Implementation summary:

- Aligned the knowledge checkpoint fixture with the current Go runtime API by keeping `cursor` as an integer and moving extended cursor detail into metadata.
- Added dashboard normalization for checkpoint lag using `cursor` and metadata `latest_source_seq`.
- Added a dashboard API for reading Go runtime job lifecycle events at `/api/dashboard/agent-jobs/{job_id}/events`.
- Added fixture-backed smoke tests for checkpoint lag, media content proxying, job event stream reads, and Python knowledge worker checkpoint cursor handling.

Tests run:

- `uv run pytest tests/test_sdd_contract_fixtures.py tests/test_knowledge_checkpoints_dashboard_plugin.py tests/test_agent_jobs_dashboard_plugin.py tests/test_dashboard_api.py::test_dashboard_media_asset_content_proxies_contract_fixture tests/test_agent_gateway_knowledge_worker.py -q --basetemp .tmp/pytest-runtime-contract-smoke`
- `go test ./...` from `services/agent-runtime`

Findings:

- The previous checkpoint fixture used an object cursor, while the implemented Go API uses an integer cursor. This slice fixed the contract drift before adding smoke coverage.
- Existing dashboard media content proxy behavior was already close to the desired boundary; the new test now pins direct proxy success, not only fallback behavior.

Decision:

- Accepted. This is still a no-external-side-effect slice and does not enable real QQ sends.

Follow-ups:

- Add the actual dashboard panel UI for worker leases, stale jobs, dead letters, checkpoint lag, and job event stream.
- Run QQ/NapCat live send smoke only after explicit approval because it sends real QQ messages.
