# SPEC-005: Go Package Structure

## Status

Accepted

## Context

The agent gateway will be implemented in Go. Its first responsibility is
backend infrastructure for QQ/Telegram channels, queue routing, bot identity,
loop protection, media events, and audit events. It should not become another
unstructured service directory.

## Decision

Use this package layout:

```text
services/agent-gateway
├── api
├── app
├── domain
├── infrastructure
├── trigger
└── types
```

## Package Responsibilities

| Package | Role | Contents |
| --- | --- | --- |
| `api` | External contract | Request/response DTOs, facade-facing schemas |
| `app` | Use-case layer | Commands, queries, application services, ports, assemblers |
| `domain` | Business core | Entities, value objects, domain services, domain errors |
| `infrastructure` | Outbound adapters | Queue, storage, NapCat, third-party API implementations |
| `trigger` | Inbound adapters | HTTP handlers, MQ listeners, jobs, webhook handlers |
| `types` | Shared primitives | Result, error code, pagination, base errors |

## Dependency Rules

Allowed:

```text
trigger        -> api, app
app            -> domain, types
infrastructure -> app, domain, types
api            -> types
domain         -> types only when unavoidable
types          -> none
```

Forbidden:

```text
domain -> app
domain -> infrastructure
domain -> trigger
domain -> api
app    -> trigger
app    -> infrastructure
api    -> app
```

## First Bounded Context

The first bounded context is `agent-gateway`:

- normalize platform messages into one envelope;
- classify human, self echo, peer bot, and system messages;
- apply explicit bot-to-bot protocol rules;
- prevent infinite bot loops;
- route inbound and outbound events through queue ports;
- write audit events for replay and SDD review.

## Acceptance Tests

- `domain` compiles without importing `app`, `trigger`, or `infrastructure`.
- an inbound HTTP trigger can call an app use case without importing domain
  models directly;
- infrastructure event bus implements an app outbound port;
- a valid ingest command publishes exactly one normalized envelope.
