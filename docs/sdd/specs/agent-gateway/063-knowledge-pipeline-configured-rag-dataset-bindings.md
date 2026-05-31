# 063 Knowledge Pipeline Configured RAG Dataset Bindings

## Context

Current Go `knowledge pipeline diagnostics` can only discover RAG datasets after a
`rag_ingest` job or `ragflow:*` checkpoint already exists. That leaves a blind spot:
Python config may already declare which datasets an observe-only QQ group should
ingest into, but Go cannot show that expectation until runtime side effects appear.

This metadata is deterministic control-plane state, not RAG strategy. It belongs in
Go diagnostics.

## Decision

Introduce configured group-level RAG dataset bindings and sync them through existing
Go-owned observe-target metadata.

### Python config

- Extend `QQGroupConfig` with optional `ragflow_dataset_ids: list[str]`.
- TOML loader accepts:
  - `ragflow_dataset_ids`
  - `ragflowDatasetIds`
- If a group does not define `ragflow_dataset_ids`, Python falls back to global
  `integrations.ragflow.default_dataset_ids`.

### Observe target sync

When Python builds observe targets for `agent_runtime`, it writes metadata:

- `channel_name`
- `ragflow_dataset_ids`: comma-separated normalized dataset ids when any exist
- `ragflow_dataset_count`: decimal count string when any exist

This remains string-only metadata so no Go API contract changes are needed.

### Go diagnostics

`knowledge pipeline diagnostics` parse configured dataset ids from observe-target
metadata and union them with runtime-discovered dataset ids from:

- `rag_ingest` jobs
- `ragflow:*` checkpoints

For a configured dataset with no runtime job/checkpoint yet:

- include it in `rag_datasets`
- `status = "muted"`
- `reasons = ["configured_dataset_not_started"]`
- `job_stage` remains empty/muted
- `checkpoint` and `checkpoint_lag` remain absent

For a configured dataset that later has runtime evidence, existing runtime-derived
status rules continue to apply.

### Totals

Pipeline totals add:

- `configured_rag_datasets`
- `configured_rag_dataset_not_started`

These count configured datasets, not just runtime-observed datasets.

## Non-goals

- No change to Python RAGFlow chunking, embedding, retrieval, or answer synthesis.
- No automatic creation of `rag_ingest` jobs purely from config metadata.
- No new Go write path for RAG dataset configuration beyond existing observe-target
  sync.

## Validation

- Python tests cover config parsing and observe-target metadata generation.
- Go tests cover configured-only datasets appearing in diagnostics before any job or
  checkpoint exists.
- Full targeted regression plus `go test ./...`.
