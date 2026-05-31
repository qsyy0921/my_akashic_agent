# 064 Knowledge Pipeline RAG Ingest Snapshot Diagnostics

## Context

Go `knowledge pipeline diagnostics` can already show:

- configured dataset bindings
- per-dataset job stage freshness
- per-dataset checkpoint lag

But after a successful `rag_ingest`, the durable checkpoint only exposes generic
metadata. The frontend or operator would need to parse raw checkpoint metadata to
understand what the last ingest actually produced.

This is a stable control-plane concern: the deterministic snapshot of a completed
ingest belongs in Go-readable runtime state, while Python keeps the actual
RAGFlow upload/parse algorithm and provider-specific details.

## Decision

Persist a bounded `rag_ingest` snapshot into checkpoint metadata and surface it
as structured Go diagnostics.

### Python worker writes

On successful non-empty `rag_ingest`, Python writes the existing checkpoint plus
these string metadata fields:

- `display_name`
- `last_message_count`
- `last_document_count`
- `last_start_seq`
- `last_end_seq`
- `last_parse_requested`

Only deterministic summary fields are persisted. Raw document ids and
provider-specific parse payload remain in Python job result detail.

### Go diagnostics read

`KnowledgePipelineRagDatasetView` gains `ingest_snapshot`, derived from durable
checkpoint metadata and `updated_at`:

- `message_count`
- `document_count`
- `start_seq`
- `end_seq`
- `parse_requested`
- `updated_at`
- `display_name`

If the checkpoint lacks these fields, `ingest_snapshot` is omitted.

### Runtime overview

Knowledge pipeline totals add:

- `rag_dataset_ingest_snapshots`

Runtime overview summary mirrors it as:

- `knowledge_pipeline_rag_dataset_ingest_snapshots`

This remains read-only. No new status transitions or side effects are introduced.

## Boundary

### Go owns

- durable checkpoint metadata interpretation
- stable HTTP/overview diagnostics shape
- aggregate counters for observability

### Python owns

- RAGFlow upload execution
- parse invocation
- document id list details
- retrieval, chunking, rerank, synthesis strategy

## Non-goals

- No change to retrieval quality or ranking logic
- No new Go write path into RAGFlow
- No durable storage of raw document id arrays in Go checkpoint metadata

## Validation

- Python tests verify worker checkpoint metadata writes
- Go tests verify `ingest_snapshot` appears in dataset diagnostics and runtime overview totals
- targeted regression plus `go test ./...`
