# Runtime Overview Control Mutation Policy

## Context

Go now exposes `GET /v1/control-mutations/policy`, a read-only view of the
supported control-plane mutation target/action allowlist. Runtime overview is
the stable aggregate used by operators and the Python dashboard, so it should
also expose the policy without requiring the dashboard to call or duplicate a
separate allowlist.

## Ownership Boundary

Go owns:

- control mutation target/action policy;
- runtime overview aggregation;
- summary/card/detail shape.

Python owns:

- optional dashboard normalization/display;
- no policy source of truth;
- no control-plane authorization.

## Goals

- Add policy dependency to runtime overview.
- Add top-level `control_mutation_policy` detail.
- Add summary fields for policy targets/actions/allowed/side effect.
- Add `Control Mutation Policy` card.
- Keep the aggregate read-only.

## Non-Goals

- No mutation execution.
- No approval or audit creation.
- No preflight mutation.
- No config/env/cutover/worker/AgentJob/MQ/outbox/AI side effects.
- No dashboard UI redesign.

## Acceptance

- Runtime overview service test covers:
  - policy summary counts;
  - policy card status/value;
  - top-level policy detail.
- `go test ./app/service` passes.
