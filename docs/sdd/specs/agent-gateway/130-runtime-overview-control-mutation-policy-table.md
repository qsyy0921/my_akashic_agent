# SPEC-130: Runtime Overview Control Mutation Policy Table

## Status

Accepted for the current iteration.

## Context

Go owns the control mutation policy allowlist and runtime overview already
aggregates it as `control_mutation_policy`. The dashboard currently falls back
to raw JSON for this detail, so operators must manually inspect payloads to see
which target/action pairs are supported before using approval and mutation
preflight APIs.

This continues OI-008 by improving dashboard readability without adding control
actions.

## Boundary Analysis

Go owns:

- control mutation policy allowlist;
- mutation target/action support checks;
- runtime overview `control_mutation_policy` detail.

Dashboard owns:

- read-only presentation of already loaded policy detail.

Out of scope:

- editing policy entries;
- creating approvals or mutation audits;
- executing cutover, worker scaling, MQ ack/nack, media cleanup, or AI work.

## Decision

When the runtime overview card id is `control_mutation_policy`, the dashboard
panel renders:

- allowed/reason/target/action totals;
- a bounded table of target kinds and actions;
- raw JSON below as fallback/debug evidence.

The renderer consumes only `card.detail` and performs no network requests.

## Acceptance

- Plugin JS asset contains `Control Mutation Policy` table sections.
- Existing runtime overview dashboard plugin tests remain green.
- SDD DONE/LIVE_CHECKS/OPEN_ISSUES/review/index are updated.
- TODO is cleared after verification.
