# ADR-0001: Use Go Message Gateway With Hexagonal Architecture

## Status

Accepted

## Context

The Python channel layer has accumulated too many responsibilities: NapCat
connection management, message filtering, media download, file handling, vision
summaries, group observation, tracing, and outbound sending. Adding multiple QQ
accounts and bot-to-bot interaction directly to that layer would increase
coupling and operational risk.

## Decision

Introduce a Go message gateway as an infrastructure boundary. The gateway uses
DDD plus hexagonal architecture with the following package layout:

- `api`: external contracts, DTOs, and facade-facing request/response models.
- `app`: application use cases, commands, queries, orchestration, and ports.
- `domain`: pure message, provenance, protocol, routing, and loop-guard logic.
- `infrastructure`: outbound adapters such as NapCat, queue, storage, and HTTP clients.
- `trigger`: inbound adapters such as HTTP handlers, MQ listeners, jobs, and webhooks.
- `types`: shared non-business primitives such as result, error code, and pagination.

The dependency direction is outer-to-inner:

```text
trigger        -> api, app
app            -> domain, types
infrastructure -> app, domain, types
api            -> types
domain         -> types only when unavoidable
types          -> no business dependency
```

Python remains responsible for Agent, LLM, tools, memory, RAG, and reasoning.

## Consequences

Positive:

- Messaging infrastructure becomes testable without LLM/runtime dependencies.
- Multiple account routing can be handled outside the Agent core.
- Bot-to-bot safety rules can be audited and tested as pure Go domain logic.
- Media, memory, and audit workers can consume queue events independently.

Negative:

- Adds one Go service and queue infrastructure.
- Requires schema discipline between Go and Python.
- Local development needs service orchestration.

## Alternatives Considered

- Continue expanding Python `QQChannel`: rejected because it worsens coupling.
- Overly strict full DDD microservice architecture: rejected as too heavy for
  the current domain. The chosen structure keeps DDD boundaries but allows a
  pragmatic Go service layout.
- Kafka-first event platform: rejected as operationally heavy at this stage.
