# Review: Phase 7.2 Python Agent Gateway Client

Spec:

- `docs/sdd/specs/agent-gateway/008-agent-job-orchestration.md`

Implementation summary:

- Added `AgentGatewayIntegrationConfig` for Python worker access to the Go
  `agent-gateway`.
- Added `integrations.agent_gateway.AgentGatewayClient` for generic job create,
  list, get, lease, running, succeeded, failed, retry, and cancel operations.
- Added compatibility tests with `httpx.MockTransport` so Python workers can be
  wired without making live HTTP calls in unit tests.
- Updated `config.example.toml` and config loading tests.
- Marked the first three AgentJob migration steps as implemented in the SDD.

Tests run:

- Pending before commit: Python agent-gateway client/config tests, Go
  `agent-gateway` tests, Go build, Python SDD/shadow regressions, and
  `git diff --check`.

Findings:

- Python had shadow mirroring into Go, but no reusable worker-side client for
  leased jobs. Without this client, image/RAG/group-memory workers would each
  grow their own HTTP glue.
- The client intentionally only owns transport and lifecycle API mapping. It
  does not execute image generation, vision, memory extraction, or RAG logic.

Decision:

- Accept as the worker compatibility layer needed before routing real
  image/RAG/group-memory execution through generic jobs.

Follow-ups:

- Convert image-generation requests to create `image_generation` jobs.
- Add a Python worker loop that leases selected job types and calls existing
  execution code.
- Add persistence before relying on jobs across gateway restarts.
