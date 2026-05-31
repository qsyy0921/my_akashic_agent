# 065 Go-Owned Knowledge Job Planner

## Context

Observe-only QQ knowledge ingestion already has a Go-owned control plane for:

- observe target persistence
- inbox capture
- AgentJob lifecycle, dedupe, lease, retry, event stream
- knowledge checkpoints
- runtime diagnostics

But recurring creation of `group_memory_extract` and `rag_ingest` jobs is still
done inside the Python knowledge worker loop. That couples deterministic job
admission to the AI worker process that should mainly lease and execute work.

This is now the main remaining control-plane gap in the observe-only knowledge
path.

## Decision

Move recurring observe-only knowledge job admission into a Go runtime worker.

### Go planner worker

Add a Go runtime worker named `knowledge_job_planner` that:

- runs on a fixed interval when explicitly enabled
- lists observe targets from Go-owned observe target state
- filters to enabled observe-only QQ group targets
- creates `group_memory_extract` jobs for each target
- creates `rag_ingest` jobs for each configured dataset binding in target metadata
- reuses the existing AgentJob dedupe behavior so repeated scans are safe

Job contract remains intentionally unchanged:

- `group_memory_extract`
  - `job_id = group_memory_extract:qq:{group_id}:{bucket}`
  - `dedupe_key = knowledge:group_memory_extract:qq:{account_id}:{group_id}`
- `rag_ingest`
  - `job_id = rag_ingest:qq:{group_id}:{dataset_id}:{bucket}`
  - `dedupe_key = knowledge:rag_ingest:qq:{account_id}:{group_id}:{dataset_id}`

Payload and metadata stay compatible with the current Python worker so lease
execution does not change.

### Python worker behavior

Python `AgentGatewayKnowledgeWorker` remains the execution worker for:

- `group_memory_extract`
- `rag_ingest`

But before running its legacy enqueue loop, it checks Go runtime config. When
Go declares `knowledge_job_planner_enabled=true`, Python skips `enqueue_once`
and only performs lease/poll execution.

If the runtime config endpoint is unavailable or does not expose the flag,
Python keeps the old enqueue fallback to preserve backward compatibility.

### Runtime diagnostics

Go runtime config adds:

- `workers.knowledge_job_planner_enabled`

Go runtime worker diagnostics add a `knowledge_job_planner` entry so the
planner is visible beside `agent_job_recovery` and `outbox_delivery_worker`.

## Boundary

### Go owns

- recurring observe-only knowledge job admission
- schedule interval and worker identity for the planner
- planner runtime diagnostics and config snapshot
- reuse of durable observe-target metadata for configured dataset bindings

### Python owns

- leasing knowledge jobs
- group memory extraction logic
- RAGFlow ingest execution
- checkpoint semantic content
- retrieval/chunking/rerank/synthesis strategy

## Non-goals

- No migration of `group_memory_extract` or `rag_ingest` execution into Go
- No change to knowledge checkpoint semantics
- No change to RAG strategy or group-memory extraction algorithm
- No generic scheduler rewrite; this slice is a focused runtime worker

## Validation

- Go unit tests cover planner job creation, dataset binding parsing, and disabled/non-QQ target filtering
- Go tests cover runtime config and runtime worker diagnostics exposure
- Python tests verify knowledge worker suppresses legacy enqueue when Go runtime advertises planner ownership
- Targeted regression plus `go test ./...`
