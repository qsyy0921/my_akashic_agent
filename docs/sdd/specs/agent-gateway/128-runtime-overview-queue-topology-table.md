# SPEC-128: Runtime Overview Queue Topology Table

## Status

Accepted for the current iteration.

## Context

Go already owns `/v1/queue-topology` and aggregates `queue_topology` into
runtime overview. The dashboard plugin currently keeps this detail visible as
raw JSON only. Operators need to inspect execution owner, ack owner, allowed
state and blockers quickly before any future QQ/NapCat or AgentJob MQ cutover.

This continues OI-008 by making another Go-owned runtime overview detail
readable without adding mutation behavior.

## Boundary Analysis

Go owns:

- queue topology read model;
- execution owner, ack owner, provider capability and blocker fields;
- runtime overview summary/card/detail source of truth.

Dashboard owns:

- read-only projection of the already loaded topology detail.

Out of scope:

- publish, lease, ack, nack, or term MQ messages;
- create, lease, retry, or cancel AgentJobs/outbox deliveries;
- start workers, modify env/config, or trigger Python AI.

## Decision

When the runtime overview card id is `queue_topology`, the dashboard panel
renders:

- provider/mode/phase/external-lease readiness summary;
- a work-kind table with queue source, execution owner, ack owner, allowed
  status and blockers;
- a node/edge count summary;
- raw JSON below as fallback/debug evidence.

The renderer consumes only `card.detail.queue_topology` when present, otherwise
`card.detail`, and performs no network requests.

## Acceptance

- Plugin JS asset contains `Queue Topology` table rendering and owner labels.
- Existing runtime overview dashboard plugin tests remain green.
- SDD DONE/LIVE_CHECKS/OPEN_ISSUES/review/index are updated.
- TODO is cleared after verification.
