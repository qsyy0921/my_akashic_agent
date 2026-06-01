# Review: Control Mutation Preflight

Spec: `docs/sdd/specs/agent-gateway/096-control-mutation-preflight.md`

## Implementation Summary

- Added Go application use case `ControlMutationPreflightService`.
- Added input port `ControlMutationPreflightChecker`.
- Added `GET /v1/control-mutations/preflight`.
- The preflight validates target/action/operator/approval inputs, reuses the
  operator approval check and returns blockers when approval is not ready.
- Ready results include a suggested `planned` mutation audit payload shape so a
  later executor can bind mutation audit records to the checked approval.

## Boundary Check

- Go owns deterministic approval lookup, preflight decision and mutation audit
  binding shape.
- Python remains responsible for AI workers and model/tool/RAG execution.
- The endpoint is read-only: it does not create approvals, create mutation
  audits, modify config/env, execute cutover, start workers, mutate AgentJobs,
  ack/nack MQ, dispatch outbox deliveries or invoke AI.

## Tests

- `go test ./app/service ./trigger/http`

## Findings

- The preflight service depends only on a minimal approval-checking interface,
  not on the full HTTP manager surface.
- Existing approval and control mutation audit endpoints remain unchanged.

## Decision

Accept.

## Follow-Ups

- A future real control-plane executor must require this preflight result,
  persist a mutation audit id, enforce rate limits/fuse rules and write rollback
  records before applying any mutation.
