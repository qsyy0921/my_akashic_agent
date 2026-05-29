# SPEC-008: Agent Job Orchestration

## Status

Partially implemented.

Implemented:

- Go `AgentJob` domain model and in-memory repository/queue.
- HTTP control-plane endpoints.
- Python compatibility client for worker leases and lifecycle updates.
- Legacy `/v1/image-jobs` create generic `image_generation` AgentJob
  compatibility records.
- Python image worker can lease generic `image_generation` jobs, execute the
  configured ChatGPT image tool, and update both generic and legacy image job
  state.
- Python knowledge worker can enqueue observe-only QQ group
  `group_memory_extract` and optional `rag_ingest` jobs, lease them from Go,
  execute group memory/RAGFlow indexing, and write lifecycle results back to
  generic jobs.

Pending:

- Add persistence and dashboard job panel.

## Context

Image generation, group-memory extraction, RAG ingestion, OCR, file indexing,
and future scheduled tasks are long-running operations. They should not block
the inbound message queue or depend on Python in-process state for retry and
visibility.

Python should execute AI-heavy workers, but Go should own job lifecycle,
leasing, retries, dead letters, and dashboard-visible state.

## Decision

Add a Go-owned `AgentJob` control plane under `services/agent-gateway` before
considering a separate job service. This follows ADR-0003: keep one deployable
service until independent deployment is justified.

Supported job types:

- `image_generation`
- `media_vision`
- `media_ocr`
- `group_memory_extract`
- `rag_ingest`
- `rag_eval`

Supported statuses:

- `pending`
- `leased`
- `running`
- `succeeded`
- `failed`
- `dead_lettered`
- `cancelled`

## Aggregate

`AgentJob` contains:

- `job_id`
- `job_type`
- `agent_id`
- `route` with platform, account id, conversation id, conversation type
- `source_event_ids`
- `source_asset_ids`
- `payload`
- `status`
- `attempts`
- `max_attempts`
- `lease_owner`
- `lease_expires_at`
- `result`
- `error_message`
- `created_at`
- `updated_at`
- `metadata`

## HTTP Contract

Create job:

```text
POST /v1/jobs
```

List recent jobs:

```text
GET /v1/jobs?limit=50&type=rag_ingest&status=pending
```

Get one job:

```text
GET /v1/jobs/{job_id}
```

Lease work:

```text
POST /v1/jobs/{job_id}/lease
POST /v1/jobs/lease-next
```

Update lifecycle:

```text
POST /v1/jobs/{job_id}/running
POST /v1/jobs/{job_id}/succeeded
POST /v1/jobs/{job_id}/failed
POST /v1/jobs/{job_id}/retry
POST /v1/jobs/{job_id}/cancel
```

## Worker Boundary

Python workers consume Go jobs through HTTP first:

```text
Go JobService -> Python Worker -> Go JobService
```

Python still owns:

- ChatGPT image generation execution;
- MiMo/vision summaries;
- embeddings and vector stores;
- RAG ranking and eval;
- group-memory extraction prompts and merge logic.

Go owns:

- job ids and idempotency;
- lease and retry;
- status visibility;
- dead-letter decisions;
- source event/asset linkage;
- dashboard job queries.

## Safety Rules

- Observe-only group jobs must not emit group-visible replies.
- Every job must include account id when tied to platform content.
- Every memory/RAG result must cite source message ids or asset ids.
- Failed jobs retry only within max attempts, then dead-letter.
- Leases must expire so crashed Python workers do not hold jobs forever.
- Job payloads must not contain raw secrets; redaction happens before model
  calls.

## Acceptance Tests

- Duplicate create request returns the existing job.
- Lease-next skips leased/running/succeeded/dead-lettered jobs.
- Expired leases can be leased again.
- Failed jobs retry until max attempts, then dead-letter.
- Cancelled jobs cannot be leased or retried.
- Group-memory and RAG job fixtures include source citations.

## Migration Plan

- [x] Add `AgentJob` domain model and in-memory repository/queue.
- [x] Add HTTP control-plane endpoints.
- [x] Add Python worker compatibility client tests.
- [x] Route image generation requests through generic jobs while preserving current
  output behavior.
  - [x] Legacy `/v1/image-jobs` creates a generic `image_generation` AgentJob.
  - [x] Python image worker consumes generic jobs and reports both generic and
    legacy image-job completion.
- [x] Route group-memory/RAG ingestion through generic jobs in observe-only mode.
  - [x] Observe-only QQ groups enqueue `group_memory_extract` jobs.
  - [x] RAGFlow-enabled observe-only QQ groups enqueue `rag_ingest` jobs for
    configured default datasets.
  - [x] Python knowledge worker consumes both job types and writes results back
    to generic jobs without group-visible replies.
- [ ] Add persistence and dashboard job panel.
