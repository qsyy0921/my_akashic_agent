# SPEC-011: Knowledge Worker Diagnostics

## Status

Implemented initial Go read-only diagnostics for memory and RAG workers.

## Context

Go now owns generic `AgentJob` lifecycle state and RAGFlow
`KnowledgeCheckpoint` cursors. Python still executes group-memory extraction
and RAGFlow uploads. Operators need one Go-owned diagnostic view that answers:

- are memory/RAG jobs pending, leased, running, failed, or dead-lettered?
- are any leases stale or leaseable again?
- what checkpoints currently gate RAG and memory ingestion?
- what recent jobs/checkpoints should be inspected before changing workers?

This is deterministic operational state, so it belongs in the Go runtime rather
than in Python worker code.

## Goals

- Add a read-only Go use case for knowledge worker diagnostics.
- Combine generic job lists with relevant knowledge checkpoints.
- Cover `group_memory_extract` and `rag_ingest` without changing Python worker
  execution.
- Keep the response safe for dashboard and operational tooling.

## Non-Goals

- Do not move group-memory extraction, prompt logic, RAGFlow upload, parsing, or
  ranking to Go.
- Do not add a new queue backend in this slice.
- Do not mutate jobs or checkpoints through the diagnostics endpoint.

## Use Case

```text
KnowledgeWorkerDiagnosticsService
├── AgentJobRepository
├── KnowledgeCheckpointRepository
└── Get(filter)
```

The service samples recent jobs for each worker type:

```text
group_memory_extract -> checkpoint prefix memory:
rag_ingest           -> checkpoint prefix ragflow:
```

For each worker type the service returns:

- recent jobs;
- status counts within the sample;
- leaseable job count;
- stale lease count;
- checkpoint list;
- latest job;
- latest checkpoint.

Stale lease detection:

- only `leased` and `running` jobs can be stale;
- a job is stale when its lease has expired, or when its `updated_at` is older
  than `stale_after_seconds`.

## HTTP API

```text
GET /v1/knowledge-worker-diagnostics?limit=50&stale_after_seconds=900
```

Response shape:

```json
{
  "generated_at": "2026-05-30T08:00:00Z",
  "stale_after_seconds": 900,
  "sampled_job_limit": 50,
  "sampled_checkpoint_limit": 50,
  "totals": {
    "jobs": 2,
    "checkpoints": 1,
    "stale_leases": 0,
    "leaseable_jobs": 1
  },
  "workers": [
    {
      "job_type": "rag_ingest",
      "checkpoint_prefix": "ragflow:",
      "status_counts": {"pending": 1},
      "stale_lease_count": 0,
      "leaseable_count": 1,
      "recent_jobs": [],
      "checkpoints": []
    }
  ]
}
```

## Acceptance

- Go app service test proves the diagnostic view summarizes
  `group_memory_extract` and `rag_ingest` jobs plus `memory:`/`ragflow:`
  checkpoints.
- Go HTTP test proves the runtime exposes
  `/v1/knowledge-worker-diagnostics`.
- Existing `AgentJob` and `KnowledgeCheckpoint` APIs remain unchanged.
- No Python worker behavior changes in this slice.
