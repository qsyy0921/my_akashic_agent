# Review: RAG eval agent job worker

Spec:

- `docs/sdd/specs/agent-gateway/008-agent-job-orchestration.md`
- `docs/sdd/specs/agent-gateway/013-rag-eval-jobs.md`
- `docs/sdd/specs/agent-architecture/006-contract-fixtures.md`

Implementation summary:

- Added reusable `eval.group_memory.runner.run_group_memory_fixture_eval` while preserving the existing CLI entrypoint.
- Added an opt-in Python `rag_eval` worker that leases Go-owned generic jobs, executes the offline group-memory fixture eval, and writes metrics to job results.
- Added `rag_eval_worker_enabled` under `integrations.agent_runtime`.
- Included `rag_eval` in Go knowledge-worker diagnostics so lifecycle state is visible with other memory/RAG workers.
- Added a shared `rag_eval` contract fixture and tests for the worker, fixture manifest, config loading, bootstrap wiring, and CLI smoke.

Tests run:

- `go test ./...` from `services/agent-runtime`
- `uv run pytest tests/test_agent_gateway_rag_eval_worker.py tests/test_sdd_contract_fixtures.py tests/test_bootstrap_wiring_p2.py::test_config_load_reads_agent_gateway_integration_block tests/test_bootstrap_wiring_p2.py::test_config_load_reads_agent_runtime_integration_block_with_compatibility tests/test_bootstrap_wiring_p2.py::test_bootstrap_runtime_rag_eval_worker_is_opt_in -q --basetemp .tmp/pytest-rag-eval-worker`
- `uv run python -m eval.group_memory.run_open_fixture_eval --workspace .tmp/group_memory_eval_smoke --fixture tests/fixtures/group_memory_open_strategy_dataset.json`

Findings:

- The Go domain model already supported `rag_eval`; the missing piece was the Python worker and documentation/config wiring.
- Quality gate failure should be stored as `passed=false` in a succeeded eval job, while worker crashes or invalid fixtures should fail the lifecycle job.

Decision:

- Accepted as a no-external-side-effect migration slice. It does not send QQ messages and does not start unless `rag_eval_worker_enabled=true`.

Follow-ups:

- Add dashboard UI surfacing `rag_eval` result summaries and trends.
- Add larger public/open fixture suites once dataset conversion is stable.
