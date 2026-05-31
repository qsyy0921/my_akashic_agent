# SPEC-025: Receiver Lease Control

## Status

Implemented as a Go-owned receiver runtime control slice.

## Context

Telegram `getUpdates` only allows one active polling consumer for a bot token.
Receiver status diagnostics can show `getupdates_conflict`, but it does not
prevent two Akashic Python processes from starting the same receiver at the
same time when they share one Go runtime.

QQ/NapCat can also suffer from duplicate local receiver processes observing the
same account/channel. The platform SDK loops remain Python-owned, but the
single-instance decision is deterministic runtime infrastructure and belongs
in Go.

## Decision

Go owns a receiver lease control plane:

```text
POST /v1/receiver-leases/acquire
POST /v1/receiver-leases/renew
POST /v1/receiver-leases/release
GET  /v1/receiver-leases
```

Python receiver adapters request a lease before starting a long-running receive
loop. A lease has a stable `receiver_id`, `holder_id`, opaque `lease_token`,
and `expires_at`. A different holder cannot acquire the same receiver until the
current lease expires or is released. Renew and release require the current
token.

This slice first gates Telegram polling. QQ receiver reporting remains
diagnostic until a later slice, because interrupting a working NapCat receive
loop has higher operational risk.

## Contract

Acquire returns a side-effect-limited control-plane result:

- `acquired=true` with `lease_token` when the caller owns the receiver lease;
- `acquired=false` with the active holder and expiry when another holder owns
  it;
- `side_effect=runtime_state_only`.

Renew extends only the active matching token. Release clears only the active
matching token. Expired leases are treated as inactive during acquire/list.

## Boundaries

Go owns:

- lease validation;
- token generation;
- active/expired decision;
- runtime overview lease counters.

Python owns:

- Telegram SDK initialization and polling;
- receiver lease heartbeat while polling is active;
- reporting suspended/failed status when a lease is denied or polling conflicts.

## Acceptance

- Go exposes acquire/renew/release/list endpoints.
- Runtime overview includes receiver lease summary fields.
- Telegram polling asks Go for a lease before `start_polling`; if denied, it
  records receiver status as suspended and does not start polling.
- Tests cover Go domain/app/http behavior, Python client calls, and Telegram
  denied/acquired lease paths.
