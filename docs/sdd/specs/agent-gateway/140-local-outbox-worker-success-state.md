# SPEC-140 Local Outbox Worker Success State

## Problem

The local outbox worker is the current recommended execution-owner path for QQ
cutover on the local `provider=local` runtime. However, a leased delivery was
being mutated twice before success:

1. `LeaseNext` already moved the delivery to `dispatching` and incremented the
   attempt counter.
2. The worker then called `MarkDispatching` again.

For `max_attempts=1` smoke items, that second mutation pushed the delivery into
`dead_lettered`, so a real successful platform send could still be recorded as
failed.

## Goal

Ensure the local outbox worker treats the lease result as the authoritative
`dispatching` transition and moves directly from:

- `LeaseNext`
- `Dispatch`
- `MarkSucceeded` or `MarkFailed`

without an extra `MarkDispatching` mutation.

## Non-Goals

- Do not change outbox domain rules for queued/dispatching/failed/retry.
- Do not enable the worker by default.
- Do not broaden cutover to rich-media cases that still fail at the platform
  boundary.

## Invariants

- `LeaseNext` remains the only transition that both acquires the lease and
  increments attempts for the worker path.
- Successful dispatch must not become `dead_lettered` solely because the worker
  re-applied the dispatching transition.
- Automatic worker success still requires a real platform send; no synthetic
  success is introduced.

## Acceptance

- Unit tests for `trigger/job` pass with the worker no longer expecting a second
  dispatching mutation.
- With `AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED=true`, runtime diagnostics show
  `execution_owner=go_local_outbox_worker`.
- A real QQ private-text outbox item can be created through `POST /v1/outbound`
  and reach `status=succeeded` automatically via the local worker.
- Returning to the default local bring-up restores the prior
  `outbox_execution_path_not_ready` cutover state.
