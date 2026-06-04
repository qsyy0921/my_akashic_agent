# Review: queue topology boundary live verifier

Spec:
- `docs/sdd/specs/agent-gateway/200-queue-topology-boundary-live-verifier.md`

Implementation summary:
- Added `scripts/verify_queue_topology_boundary.py` and
  `scripts/verify-queue-topology-boundary.ps1` to read
  `/v1/queue-backend`, `/v1/queue-topology`, and `/v1/runtime-overview`,
  then emit structured JSON for provider, execution-owner, ack-owner, and
  external-lease parity.
- Updated `scripts/verify-go-migration-goal.ps1` to include the verifier output
  as `queue_topology`, reuse its conclusion for
  `residual_classification.queue_topology_boundary`, and require
  `dashboard_read_models.queue_topology_table=true`.
- Fixed the runtime-overview panel empty-state copy for `Queue Topology` to use
  the explicit literal `No queue topology work kinds sampled`, so the static
  panel asset check matches the actual rendered drilldown.

Tests run:
- `uv run pytest tests/test_verify_queue_topology_boundary.py -q`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py tests/test_verify_queue_topology_boundary.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-queue-topology-boundary.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

Findings:
- Current live runtime is internally consistent on queue topology boundaries:
  `provider=local`, `recommended=nats_jetstream`,
  `outbox_execution_owner=go_local_outbox_worker`,
  `agent_job_execution_owner=python_ai_worker_state_store_lease`,
  `agent_job_ack_owner=go_state_store_api`, and `external_lease_ready=false`
  all match across queue-backend, queue-topology, and runtime-overview summary.
- The remaining gap is not queue-topology visibility. It is still the
  production cutover flags and provider change required for
  `agent_job external lease result-ack`.

Decision:
- Accept.

Follow-ups:
- Reuse this verifier before any future queue-provider, external-lease, or
  result-ack owner cutover.
