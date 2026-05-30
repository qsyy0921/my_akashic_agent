# Review: Phase 8.22 Knowledge Worker Diagnostics

Spec:
- `docs/sdd/specs/agent-gateway/011-knowledge-worker-diagnostics.md`

Implementation summary:
- Added a Go `KnowledgeWorkerDiagnosticsService` that combines recent
  `AgentJob` rows and `KnowledgeCheckpoint` rows for `group_memory_extract` and
  `rag_ingest`.
- Added stale lease and leaseable job counts without changing worker lifecycle
  behavior.
- Added `/v1/knowledge-worker-diagnostics` as a read-only runtime endpoint.
- Wired the service in `cmd/agent-runtime` next to the existing generic job and
  knowledge checkpoint services.
- Updated the runtime README and migration TODO.

Tests run:
- `go test ./...` under `services/agent-runtime`
- `go build -o ..\..\.tmp\bin\agent-runtime-new.exe .\cmd\agent-runtime`
- Live smoke against `127.0.0.1:8780` after restart:
  `GET /v1/knowledge-worker-diagnostics?limit=5&stale_after_seconds=60`
  returned `worker_count=2` with job/checkpoint totals.
- `git diff --check -- services\agent-runtime docs\sdd`

Decision:
- Approved as an operational diagnostics slice. Go owns the deterministic
  lifecycle view; Python remains the executor for group-memory extraction and
  RAGFlow upload/parse logic.

Follow-ups:
- Add a dashboard panel that consumes this endpoint and highlights stale leases,
  dead letters, and checkpoint lag.
- Add Go/Python contract fixtures for diagnostics payload stability.
