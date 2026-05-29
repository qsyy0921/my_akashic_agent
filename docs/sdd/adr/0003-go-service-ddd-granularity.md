# ADR-0003: Go Service DDD Granularity

## Status

Accepted

## Context

The project is moving backend infrastructure from Python into Go. The main
candidate areas are message gateway, outbox, media registry, job lifecycle,
scheduler infrastructure, routing, and audit.

The open question is whether all Go code should share one global DDD package
layout or whether each Go service should own its own DDD + hexagonal structure.

## Decision

Use one DDD + hexagonal structure per Go service.

All Go services live under:

```text
services/
```

Each service owns its own layers:

```text
services/<service-name>/
├── api
├── app
├── domain
├── infrastructure
├── trigger
└── types
```

The current implementation remains a single Go service:

```text
services/message-gateway
```

Within this service, message routing, loop guard, outbox, shadow audit, image
jobs, and media registry are treated as bounded contexts inside one deployable
until they show independent deployment or persistence needs.

## Rationale

- DDD boundaries should follow bounded contexts, not a global source tree.
- A global `domain/app/infrastructure` shared by every service becomes a dumping
  ground and weakens dependency rules.
- Starting with one Go service avoids premature microservice complexity.
- Splitting later is easier when each bounded context already exposes app ports
  and DTO contracts.

## Rules

- Do not place Go code outside `services/`.
- Do not create a new Go service unless it has an independent deployment,
  scaling, persistence, or operational lifecycle reason.
- Shared code must remain minimal and generic; business types stay inside the
  owning service.
- If `services/message-gateway` becomes too broad, split by bounded context:
  `job-service`, `scheduler-service`, or `asset-service`.

## Consequences

Positive:

- The current code stays simple enough for local development.
- Architecture tests remain service-local and easy to enforce.
- Future service splits have a clear extraction path.

Negative:

- `message-gateway` will temporarily host several infrastructure contexts.
- Cross-context naming discipline is required until independent services are
  justified.
