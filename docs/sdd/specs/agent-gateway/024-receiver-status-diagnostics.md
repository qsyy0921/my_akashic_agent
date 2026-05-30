# SPEC-024: Receiver Status Diagnostics

## Status

Implemented as a Go-owned runtime diagnostics slice.

## Context

QQ and Telegram receive loops still run in Python because they depend on
platform SDK behavior and live login state. Go already owns delivery adapter
health, inbox metrics, observe targets, and runtime overview, but it does not
yet have a source-of-truth view of whether each Python receiver is actually
connected, suspended, or failed.

This gap is visible when Telegram `getUpdates` enters conflict mode: Go
`getMe` remains healthy because Bot API auth works, while Python polling is
suspended and no Telegram messages will be received.

## Decision

Python remains the platform receiver adapter. It reports receiver lifecycle
state to Go whenever a receiver starts, fails, is suspended, or stops:

```text
POST /v1/receiver-statuses/report
GET  /v1/receiver-statuses
GET  /v1/runtime-overview
```

Go owns validation, state normalization, aggregation, and dashboard-facing
runtime semantics. This slice is diagnostic only: it does not start or stop
receivers, acquire Telegram polling locks, reconnect QQ, or send platform
messages.

## Contract

Each reported receiver includes:

- `receiver_id`: stable id such as `qq:1049511700:qq` or
  `telegram:7689386159:telegram`;
- `kind`: `qq`, `telegram`, or another platform kind;
- `channel_name`: local channel alias used by Python routing;
- `account_id`: platform bot account id when known;
- `endpoint`: receiver endpoint, for example OneBot WebSocket URI;
- `status`: `starting`, `connected`, `suspended`, `failed`, or `stopped`;
- `reason`, `last_error`, `source`, `metadata`, and `updated_at`;
- `side_effect=none` on aggregate responses.

Go aggregates receiver totals by status and platform kind. Runtime overview
adds a `Receiver Statuses` card and summary counters so operators can
distinguish "adapter can send" from "receiver can currently observe".

## Boundaries

Go owns:

- receiver status validation;
- latest-status runtime storage;
- totals and runtime overview card semantics.

Python owns:

- live QQ/NcatBot and Telegram polling loops;
- detecting start success/failure and Telegram conflict suspension;
- best-effort status reporting to Go.

## Acceptance

- Go exposes report/list endpoints for receiver statuses.
- Python reports QQ start success/failure and Telegram start/conflict status.
- Runtime overview includes receiver status summary fields and card details.
- Tests cover Go domain/app/http contracts, Python client calls, and dashboard
  normalization.
