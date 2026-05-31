# 070 Knowledge Pipeline RAG Index State Diagnostics

## Context

Go already surfaces per-dataset RAG checkpoints and the last successful
`rag_ingest` snapshot metadata written by Python. That shows what was ingested,
but operators still need one stable field that answers whether a configured
dataset currently has usable indexed content.

Calling RAGFlow or any embedding/search provider from Go would cross the AI
runtime boundary. The safer control-plane step is to derive index state from
the checkpoint metadata that Python already writes after a successful ingest.

## Decision

Add a read-only `rag_index_state` to each
`KnowledgePipelineRagDatasetView`.

The state is derived only from Go-owned checkpoint metadata and source sequence
diagnostics:

- no snapshot: `status=muted`, `reason=no_ingest_snapshot`;
- snapshot with `document_count <= 0`: `status=warn`,
  `reason=no_index_documents`;
- snapshot with source lag: `status=warn`,
  `reason=index_lagging_source_seq`;
- snapshot with documents and no lag: `status=ok`,
  `reason=index_ready`.

The state exposes document count, message count, indexed seq range, last ingest
time, age seconds, source seq lag, parse requested flag, and a boolean `ready`.

Aggregate totals are added to knowledge pipeline diagnostics and runtime
overview summary:

- `rag_dataset_index_ready`;
- `rag_dataset_index_missing_snapshot`;
- `rag_dataset_index_empty`;
- `rag_dataset_index_lagging`.

## Boundary

### Go owns

- deterministic index-state derivation from checkpoint metadata;
- aggregation into knowledge pipeline diagnostics and runtime overview;
- HTTP/dashboard-safe read-only fields.

### Python owns

- RAGFlow upload and parse;
- chunking, embedding, rerank, query rewrite, answer synthesis;
- provider-specific fallback and raw external index inspection;
- writing successful `rag_ingest` checkpoint metadata.

## Non-goals

- No direct RAGFlow API calls from Go.
- No external document-count fetch from vector stores.
- No mutation of AgentJob, checkpoint, dataset or provider state.
- No change to Python RAG execution.

## Validation

- Knowledge pipeline tests cover ready, empty, missing and lagging index states.
- Runtime overview tests cover new summary fields.
- `go test ./...`, `go build ./cmd/agent-runtime`, and `git diff --check` must
  pass.
