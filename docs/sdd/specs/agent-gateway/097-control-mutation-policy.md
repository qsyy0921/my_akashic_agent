# Control Mutation Policy

## Context

`GET /v1/control-mutations/preflight` now verifies that a proposed
control-plane mutation has an active approved operator approval. That is not
enough for a future executor: an arbitrary `target_kind/action` pair should not
become executable simply because an approval exists.

This slice adds a deterministic Go-owned allowlist for supported mutation
intents. Preflight remains read-only, but it now rejects unsupported
target/action pairs before returning `ready=true`.

## Ownership Boundary

Go owns:

- supported control-plane mutation target/action policy;
- preflight gating and blocker reasons;
- stable runtime API behavior.

Python owns:

- AI worker execution;
- model/provider/prompt/tool/RAG/image workflows;
- no mutation target/action authorization.

## Goals

- Add a domain service for control mutation policy.
- Keep the policy small and explicit.
- Integrate the policy into `ControlMutationPreflightService`.
- Return stable blockers:
  - `unsupported_control_mutation_target`;
  - `unsupported_control_mutation_action`.
- Preserve read-only side effects.

## Initial Supported Intents

```text
outbound_cutover: enable, rollback
knowledge_job_planner_cutover: enable, rollback
agent_job_priority: apply, rollback
agent_job_capacity: apply, rollback
agent_job_external_lease: enable, rollback
```

These are policy labels for future control-plane executors. This slice does not
execute any of them.

## Non-Goals

- No mutation execution.
- No approval creation.
- No mutation audit creation.
- No config/env/cutover changes.
- No AgentJob, MQ, worker, outbox or AI side effects.
- No runtime overview or dashboard redesign.

## Acceptance

- Domain policy tests cover supported target/action and unsupported cases.
- Preflight service tests cover unsupported action.
- HTTP handler tests cover unsupported target/action response.
- `go test ./domain/service ./app/service ./trigger/http` passes.
