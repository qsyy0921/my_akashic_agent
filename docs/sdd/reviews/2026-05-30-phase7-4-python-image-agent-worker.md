# Review: Phase 7.4 Python Image Agent Worker

Spec:

- `docs/sdd/specs/agent-gateway/008-agent-job-orchestration.md`

Implementation summary:

- Added `AgentGatewayImageWorker` to lease generic `image_generation` jobs from
  Go.
- The worker executes the existing `ChatGPTImageGenerateTool`, converts saved
  image paths into image attachments, and completes both the generic
  `AgentJob` and the legacy `ImageJob`.
- Added legacy `/v1/image-jobs/{id}` state update methods to
  `AgentGatewayClient`.
- Added app startup wiring gated by `integrations.agent_gateway.enabled` and
  `integrations.chatgpt_proxy.enabled`.
- Added tests for success, no-job idle behavior, and failure propagation to both
  generic and legacy job states.

Tests run:

- `uv run pytest tests\test_agent_gateway_client.py tests\test_agent_gateway_image_worker.py tests\test_bootstrap_wiring_p2.py::test_config_load_reads_agent_gateway_integration_block tests\test_sdd_contract_fixtures.py tests\test_shadow_gateway.py -q --basetemp .tmp\pytest-phase7-4-run`
- `go test ./...` under `services/agent-gateway`
- `go build ./cmd/agent-gateway` under `services/agent-gateway`
- `python -m compileall integrations\agent_gateway.py integrations\agent_gateway_image_worker.py bootstrap\app.py agent\config.py agent\config_models.py`
- `git diff --check`
- Live gateway smoke: create legacy `/v1/image-jobs`, run
  `AgentGatewayImageWorker.process_once()`, then verify both generic
  `/v1/jobs/{job_id}` and legacy `/v1/image-jobs/{job_id}` are `succeeded`.

Findings:

- The worker is intentionally gated by config. This avoids starting a polling
  worker when the ChatGPT proxy is unavailable or the project is running in the
  old direct-tool mode.
- Sending the generated image back to the platform remains the responsibility
  of the existing message/outbox path; this slice records the generated
  attachment and job completion.

Decision:

- Accept as the first real Python worker consuming Go-owned generic jobs.

Follow-ups:

- Add a platform delivery step that turns completed image attachments into
  outbound messages when the originating request expects a direct reply.
- Route group-memory and RAG ingestion through generic jobs in observe-only
  mode.
- Add durable job persistence before relying on jobs across gateway restarts.
