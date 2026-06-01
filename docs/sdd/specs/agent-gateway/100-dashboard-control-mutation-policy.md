# Dashboard Control Mutation Policy

## Context

Go runtime overview now exposes `control_mutation_policy`, including summary
fields and a `Control Mutation Policy` card. The Python dashboard runtime
overview plugin should normalize this Go-owned detail so the browser API gets a
stable shape without duplicating the allowlist or owning control-plane
authorization.

## Ownership Boundary

Go owns:

- control mutation target/action allowlist;
- policy query API;
- runtime overview aggregation.

Python owns:

- read-only dashboard normalization;
- no local policy source of truth;
- no approval, mutation or execution authority.

## Goals

- Normalize `control_mutation_policy` in `/api/dashboard/runtime-overview`.
- Add summary defaults:
  - `control_mutation_policy_allowed`;
  - `control_mutation_policy_reason`;
  - `control_mutation_policy_targets`;
  - `control_mutation_policy_actions`.
- Preserve the `control_mutation_policy` card from Go.
- Keep all side effects read-only.

## Non-Goals

- No mutation execution.
- No approval creation.
- No mutation audit creation.
- No config/env/cutover/worker/AgentJob/MQ/outbox/AI side effects.
- No dashboard UI redesign.

## Acceptance

- Dashboard plugin test verifies:
  - `control_mutation_policy` card is present;
  - summary defaults expose policy allowed/reason/target/action fields;
  - top-level `control_mutation_policy` detail has stable intents and
    `side_effect=none`.
- Targeted Python dashboard test passes.
