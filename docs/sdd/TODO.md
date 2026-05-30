# Akashic Go Migration TODO

Last updated: 2026-05-30

## Done

- [x] Rename the Go service boundary to `services/agent-runtime`.
- [x] Keep Go code under `services/agent-runtime` with DDD + hexagonal package
  boundaries.
- [x] Add Go-owned shadow audit ingestion and read APIs.
- [x] Add Go-owned media asset registry and safe content access.
- [x] Add dashboard media links through Go media asset routes.
- [x] Add Go-owned send ledger for recent-send and echo-loop protection.
- [x] Add Go-owned outbox lifecycle, persistence, leasing, retry, and failure
  classification.
- [x] Add Go-owned generic agent job lifecycle and persistence.
- [x] Move observe-only group-memory extraction lifecycle to Go generic jobs
  while Python remains the extraction worker.
- [x] Move group-memory and RAGFlow source replay to the Go inbox API.
- [x] Add Go-owned knowledge checkpoints for RAGFlow `rag_ingest` cursors.
- [x] Fix Go runtime JSON responses to declare UTF-8 for PowerShell clients.
- [x] Add dashboard visibility for Go-owned knowledge checkpoints.
- [x] Fix Go media content safe-root discovery for Windows local runtime starts
  so QQ image/file attachments open from the dashboard.
- [x] Add Go-owned generic job checkpoint/list diagnostics for memory and RAG
  workers.

## Next

- [ ] Add a Go-owned queue adapter or durable stream behind current in-process
  job leasing once local file-backed state is stable.
- [ ] Move proactive delivery scheduling state into Go while keeping prompt and
  decision generation in Python.
- [ ] Move Telegram/QQ outbound dispatch behind a Go `DeliveryAdapter` after the
  current Python compatibility outbox worker is stable.
- [ ] Add Go/Python contract fixtures for checkpoint, inbox replay, outbox
  delivery, and media asset content routes.
- [ ] Add RAG evaluation jobs as Go-owned lifecycle records with Python eval
  workers.
- [ ] Add operational dashboard panels for runtime health, worker leases, stale
  jobs, dead letters, and checkpoint lag.

## Guardrails

- Go owns deterministic infrastructure: routing state, idempotency, durable
  stores, lifecycle, retries, leases, checkpoints, assets, and audit.
- Python owns AI behavior: model calls, prompts, RAGFlow uploads, OCR/VLM,
  extraction heuristics, ranking, generation, and fast experiments.
- Every completed migration slice must update this TODO plus the relevant SDD
  spec/review files.
