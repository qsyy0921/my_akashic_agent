# Phase 8.87 Review: Agent Job Admission Dedupe

## Scope

- Added optional `dedupe_key` to `POST /v1/jobs`.
- Go `AgentJobService.Create` now checks for an existing active job of the same
  type and dedupe key before creating a new job.
- Active states are `pending`, `leased`, and `running`; terminal jobs no longer
  suppress future creates.
- File-backed and memory repositories implement the same active dedupe lookup.
- Python `AgentGatewayClient.create_job` can pass `dedupe_key`.
- The observe-only knowledge worker now submits stable dedupe keys for
  `group_memory_extract` and `rag_ingest` jobs and reports
  `suppressed_by_runtime_dedupe`.

## Boundary Check

- Queue admission, lifecycle event suppression, and work notification
  suppression are deterministic runtime infrastructure, so they belong in Go.
- Python still decides which observe-only groups/datasets are scheduled and
  executes group-memory/RAG work after leasing.
- The change does not send QQ/Telegram messages, call models, download files, or
  alter group observation behavior.

## Dedupe Keys

```text
knowledge:group_memory_extract:qq:{account_id}:{group_id}
knowledge:rag_ingest:qq:{account_id}:{group_id}:{dataset_id}
```

## Tests

- Go service test proves duplicate active creates return the existing job and
  do not append an extra created event.
- Go HTTP test proves `dedupe_key` on `/v1/jobs` returns the active existing
  job id.
- Python client test proves `dedupe_key` is serialized.
- Python knowledge worker tests prove stable dedupe keys are passed and runtime
  suppression is surfaced in enqueue summaries.

## Risks

- Dedupe lookup scans current repository state. This is acceptable for the
  current single-node file/memory store and keeps the design inside
  `agent-runtime`; a future database-backed store should index `dedupe_key`.
- A pending job can suppress later schedule buckets until it is leased,
  cancelled, or completed. That is intentional backpressure for observe-only
  memory/RAG jobs.

## Decision

Accept as the first Go-owned queue admission guard for generic AgentJob
scheduling.
