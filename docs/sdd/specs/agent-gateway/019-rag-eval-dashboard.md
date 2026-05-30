# SPEC-019: RAG Eval Dashboard

## Status

Implemented initial read-only quality-gate dashboard plugin.

## Context

`rag_eval` jobs are owned by Go generic job lifecycle and executed by an
opt-in Python evaluation worker. The previous runtime overview shows only a
failure count. Operators need a dedicated view that can inspect evaluation
metrics, quality gate failures, and trend points without reading raw job JSON.

## Goals

- Read Go-owned `rag_eval` jobs from the generic job endpoint.
- Parse worker result fields robustly because Go stores job result values as
  strings.
- Distinguish quality failures (`passed=false` on a succeeded job) from
  infrastructure failures (`failed`, `dead_lettered`, or `cancelled` lifecycle
  state).
- Expose a compact dashboard table and detail view for per-question results.
- Keep the plugin read-only.

## Runtime Reads

The plugin reads:

```text
GET /v1/jobs?type=rag_eval
```

Result fields parsed from each job:

```text
questions
top1_accuracy
evidence_coverage
min_top1_accuracy
min_evidence_coverage
passed
results
```

`results` may be a JSON string containing per-question details.

## Quality Status

The dashboard derives a normalized `quality_status`:

- `passed`: lifecycle succeeded and `passed=true`;
- `failed_quality`: lifecycle succeeded and `passed=false`;
- `infra_failed`: lifecycle is `failed`, `dead_lettered`, or `cancelled`;
- `pending`, `leased`, `running`: active lifecycle state;
- `unknown`: metrics are incomplete.

## Boundaries

Go owns:

- job persistence, lifecycle state, leasing, retry/dead-letter status, and event
  stream.

Python owns:

- evaluation execution and metric production.

Dashboard owns:

- read-only metric parsing, summary, trend shaping, and UI rendering.

## Acceptance

- `/api/dashboard/rag-eval` returns paged jobs, summary, and trend points.
- The panel is discoverable via `/api/dashboard/plugins`.
- Tests cover pass rate, quality failure, infrastructure failure, active job
  count, trend extraction, and per-question result parsing.
