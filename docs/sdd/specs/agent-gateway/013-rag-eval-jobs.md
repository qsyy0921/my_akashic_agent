# SPEC-013: RAG Evaluation Jobs

## Status

Implemented initial Go-owned lifecycle with a Python eval worker.

## Context

RAG and group-memory quality gates should be repeatable jobs instead of ad hoc
local commands. The evaluation itself belongs in Python because it uses memory
stores, retrieval code, fixtures, metrics, and future ML tooling. Lifecycle,
leasing, retries, visibility, and event streams belong in Go.

## Goals

- Use the existing Go `AgentJob` aggregate for `rag_eval`.
- Keep evaluation execution in Python workers.
- Preserve the existing offline group-memory eval CLI.
- Write evaluation metrics back to the generic job result.
- Make the worker opt-in so normal QQ observation does not trigger eval jobs.

## Job Contract

Job type:

```text
rag_eval
```

Payload fields:

```text
suite                   group_memory_open_fixture
fixture                 tests/fixtures/group_memory_open_strategy_dataset.json
workspace               optional isolated eval workspace
min_top1_accuracy       threshold, default 1.0
min_evidence_coverage   threshold, default 1.0
```

Result fields are stringified for the generic Go job result map:

```text
questions
top1_accuracy
evidence_coverage
min_top1_accuracy
min_evidence_coverage
passed
results
```

`passed=false` is a quality result, not an infrastructure failure. The worker
marks the job succeeded when evaluation executes and writes the metrics. The job
is failed only when the evaluator crashes, the fixture is invalid, or the worker
cannot run.

## Runtime Configuration

```toml
[integrations.agent_runtime]
enabled = true
rag_eval_worker_enabled = true
```

When enabled, Python starts a worker that leases only `rag_eval` jobs:

```text
POST /v1/jobs/lease-next {"job_type":"rag_eval"}
POST /v1/jobs/{job_id}/running
POST /v1/jobs/{job_id}/succeeded
POST /v1/jobs/{job_id}/failed
```

## Boundaries

Go owns:

- `rag_eval` job ids, routes, source ids, leases, retries, status, and events.
- Dashboard/job diagnostics and durable lifecycle stream.

Python owns:

- fixture loading;
- group-memory eval execution;
- retrieval metrics and quality thresholds;
- future ML/RAG benchmark integrations.

## Acceptance

- Go accepts, leases, and records lifecycle events for `rag_eval`.
- Python worker can lease a `rag_eval` job, execute the checked-in fixture, and
  complete the job with metrics.
- The original CLI `python -m eval.group_memory.run_open_fixture_eval` still
  works.
- The shared contract fixture manifest includes a `rag_eval` job fixture.
