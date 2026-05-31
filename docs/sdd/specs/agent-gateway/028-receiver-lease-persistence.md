# Receiver Lease Persistence

## Status

Accepted

## Problem

Receiver lease control prevents duplicate long-running platform polling loops,
but the first implementation kept leases in memory. Restarting only
`agent-runtime` removed the active Telegram lease while the Python Telegram
polling process kept running. The next renew then failed with
`receiver lease not found`, leaving single-owner diagnostics unreliable until a
full Python restart.

## Decision

Go owns durable receiver lease state:

- unconfigured receiver lease storage defaults to
  `.akashic-workspace/agent-runtime/receiver-leases.json`;
- `AKASHIC_RECEIVER_LEASES_DSN` and `AKASHIC_RECEIVER_LEASES_PATH` can override
  the file path;
- `AKASHIC_RECEIVER_LEASES_DSN=memory` keeps the legacy ephemeral behavior for a
  specific run;
- receiver status and receiver lease stores remain separate files because status
  is observational, while lease tokens are control-plane fencing state.

On startup, `agent-runtime` loads persisted receiver leases. Renew continues to
require the matching `holder_id` and `lease_token`, and now rejects expired
leases explicitly with `receiver lease expired`. Acquire still treats expired
leases as inactive and can issue a new token.

Python Telegram polling keeps the SDK loop Python-owned, but renew failure now
has a guarded recovery path:

- `receiver lease not found`, `receiver lease expired`, and
  `receiver lease token mismatch` trigger a reacquire attempt;
- if reacquire succeeds, renewal continues with the new token;
- if another holder owns the lease, Telegram polling is suspended and receiver
  status is reported as `suspended` with
  `reason=receiver_lease_held_after_renew`.

## Boundaries

- Go still does not run Telegram polling in this slice.
- Receiver lease recovery does not send QQ/Telegram messages.
- This slice only hardens Telegram lease ownership. QQ receiver leases remain a
  later cutover decision after observe-only image/file coverage is stable.

## Acceptance

- Receiver leases survive `agent-runtime` restart through the file-backed
  repository.
- Renewing a persisted active lease with the same token succeeds after reopening
  the service.
- Renewing an expired lease fails with `receiver lease expired`, allowing Python
  to reacquire instead of silently extending stale ownership.
- Python Telegram attempts reacquire on missing/expired/token-mismatch renew
  errors and suspends polling if the lease is now held by another holder.
- Runtime config diagnostics expose receiver lease persistence override
  environment variables.
