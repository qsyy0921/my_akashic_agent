# SPEC-010: Knowledge Checkpoints

## Status

Implemented initial RAGFlow cursor slice and dashboard visibility.

## Context

Go now owns durable inbox replay and generic `rag_ingest` jobs, but Python still
needs a stable cursor to avoid repeatedly uploading the same QQ group messages
to external RAG systems. That cursor is operational infrastructure, not AI
reasoning logic, so it belongs in the Go runtime.

## Goals

- Persist per-source/per-target knowledge ingestion cursors in Go.
- Prevent cursor regression after successful ingestion.
- Expose a small HTTP control plane for Python workers.
- Keep Python responsible for RAGFlow upload/parse behavior.

## Domain Model

```text
KnowledgeCheckpoint
├── checkpoint_id
├── cursor
├── updated_at
└── metadata
```

Invariants:

- `checkpoint_id` is required.
- `cursor >= -1`.
- `updated_at` is required.
- `Advance` rejects moving the cursor backwards.

Checkpoint id convention for RAGFlow QQ group ingest:

```text
ragflow:qq:{group_id}:{dataset_id}
```

## Ports

Inbound app port:

```text
KnowledgeCheckpointManager
├── List(filter)
├── Get(checkpoint_id)
└── Upsert(checkpoint_id, cursor, metadata)
```

Outbound app port:

```text
KnowledgeCheckpointRepository
├── SaveKnowledgeCheckpoint(checkpoint)
└── FindKnowledgeCheckpoint(checkpoint_id)
└── ListKnowledgeCheckpoints(filter)
```

## HTTP API

```text
GET /v1/knowledge-checkpoints?limit=50&prefix=ragflow:qq:
GET /v1/knowledge-checkpoints/{checkpoint_id}
PUT /v1/knowledge-checkpoints/{checkpoint_id}
```

PUT body:

```json
{
  "cursor": 2559,
  "metadata": {
    "job_type": "rag_ingest",
    "source": "qq",
    "group_id": "27234224",
    "dataset_id": "ds1"
  }
}
```

## Runtime Configuration

```powershell
$env:AKASHIC_KNOWLEDGE_CHECKPOINTS_DSN = "E:\agent\akashic\.akashic-workspace\runtime\knowledge-checkpoints.json"
```

`AKASHIC_KNOWLEDGE_CHECKPOINTS_PATH` is accepted as a shorthand. The special
value `memory` keeps the development in-memory store.

## Acceptance

- RAGFlow `rag_ingest` reads the checkpoint before choosing `since_seq`.
- Successful ingest with messages advances the checkpoint to the uploaded
  `end_seq`.
- Empty ingest succeeds without advancing the checkpoint.
- Dashboard can list and inspect Go-owned checkpoints through the runtime API.
- Go tests cover domain regression protection, app service behavior, HTTP API,
  and file-backed persistence.

## Non-Goals

- Do not move RAGFlow API upload/parse logic to Go.
- Do not solve deduplication inside RAGFlow datasets in this slice.
