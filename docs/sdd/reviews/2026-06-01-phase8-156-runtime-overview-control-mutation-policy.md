# Review: Runtime Overview Control Mutation Policy

Spec: `docs/sdd/specs/agent-gateway/099-runtime-overview-control-mutation-policy.md`

## Implementation Summary

- Added `ControlMutationPolicy` dependency to runtime overview.
- Added top-level `control_mutation_policy` detail to `RuntimeOverviewView`.
- Added summary fields:
  - `control_mutation_policy_allowed`;
  - `control_mutation_policy_reason`;
  - `control_mutation_policy_targets`;
  - `control_mutation_policy_actions`.
- Added `Control Mutation Policy` card with target/action count.
- Updated runtime overview tests for summary, card and detail.

## Boundary Check

- Go remains the source of truth for mutation policy and runtime aggregation.
- Python/dashboard can consume overview detail but does not maintain the
  allowlist.
- The aggregate remains read-only and does not create approvals, create mutation
  audits, mutate config/env, execute cutover, start workers, mutate AgentJobs,
  ack/nack MQ, dispatch outbox deliveries or invoke AI.

## Tests

- `go test ./app/service`

## Findings

- Runtime overview now exposes the control-plane mutation surface in the same
  stable API as control audit and cutover plans.
- Future dashboard work can render the overview detail without directly calling
  the policy endpoint.

## Decision

Accept.

## Follow-Ups

- Optional Python dashboard normalization can be added if frontend code needs
  flattened defaults for `control_mutation_policy`.
