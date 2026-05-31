# Phase 8.135 AgentJob Capacity Plan

Spec: `docs/sdd/specs/agent-gateway/078-agent-job-capacity-plan.md`

## Changes

- Added `GET /v1/agent-job-capacity/plan`.
- Added Go `AgentJobCapacityPlanService` under `services/agent-runtime/app/service`.
- Added command/query/port types for a read-only capacity plan.
- The plan combines Go-owned AgentJob pressure and Python worker heartbeat
  coverage into operational recommendations:
  - `start_or_recover_python_worker`
  - `inspect_failed_worker`
  - `renew_or_restart_stale_worker`
  - `increase_worker_concurrency_or_prioritize_queue`
  - `monitor`
- The endpoint is advisory only. It does not lease/retry jobs, start workers,
  acknowledge MQ messages, mutate config, or execute AI work.

## Boundary Review

- Go owns deterministic control-plane diagnostics, recommendation labels,
  HTTP contract and SDD/test coverage.
- Python remains responsible for actual AI worker lifecycle, concurrency
  implementation, MiMo/model calls, group memory extraction, RAG/chunking,
  embedding/rerank, OCR/VLM, image generation, prompt and tool execution.
- This slice deliberately avoids autoscaling, priority scheduling, or queue
  acknowledgement behavior.

## Verification

- `go test ./app/service -run TestAgentJobCapacityPlan -count=1 -v`
- `go test ./trigger/http -run TestAgentJobCapacityPlanEndpointReturnsReadOnlyPlan -count=1 -v`
- `go test ./cmd/agent-runtime -run TestRuntimeConfig -count=1 -v`
- `go test ./...`
- `go build ./cmd/agent-runtime`

## Residual Risk

- Live behavior still depends on Python workers reporting accurate
  `agent-worker-statuses`.
- The plan does not decide concrete replica counts or concurrency values; that
  should be a later explicit control-plane design with operator acknowledgement
  and audit.
