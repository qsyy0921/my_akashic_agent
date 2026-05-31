# Review: Knowledge Pipeline RAG Index State

Spec:

- `docs/sdd/specs/agent-gateway/070-knowledge-pipeline-rag-index-state-diagnostics.md`

Implementation summary:

- Added `rag_index_state` to each `KnowledgePipelineRagDatasetView`.
- Derived ready/missing snapshot/empty index/source-lagging states from existing
  checkpoint snapshot metadata.
- Added knowledge pipeline totals for index ready, missing snapshot, empty and
  lagging datasets.
- Added runtime overview summary fields for those totals.
- Updated service and HTTP tests to cover ready, configured-only missing,
  empty-index and lagging-index cases.

Tests run:

- `go test ./app/service -run "TestKnowledgePipelineDiagnosticsService|TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics" -count=1 -v`
- `go test ./trigger/http -run "TestKnowledgePipelineDiagnosticsEndpointReturnsReadOnlyPipelines|TestRuntimeOverviewEndpointReturnsGoOwnedAggregate" -count=1 -v`
- `go test ./...`
- `go build ./cmd/agent-runtime`
- `git diff --check`

Findings:

- Go can now explain whether a dataset has usable indexed content without
  calling RAGFlow or inspecting provider-specific state.
- Empty `document_count` now warns the dataset and parent pipeline.
- Missing snapshots stay muted for configured-only datasets, preserving the
  distinction between "configured but not started" and "ran but produced no
  usable documents".

Decision:

- Accepted pending full regression. The change improves Go-owned RAG
  observability while keeping Python responsible for actual indexing and AI
  strategy.

Follow-ups:

- If Python later writes provider-stable parse/index metadata, Go can extend
  `rag_index_state` without changing the external RAG provider boundary.
