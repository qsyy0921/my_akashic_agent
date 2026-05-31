# Receiver Status Heartbeat

## Status

Accepted

## Problem

`GET /v1/observe-capture-diagnostics` depends on receiver connectivity. The
previous receiver status store was in-memory only and Python reported receiver
state mainly at startup or on exceptional paths. Restarting only
`agent-runtime` therefore cleared receiver status while QQ/NapCat receivers kept
collecting messages, producing a misleading `receiver_not_connected` blocker.

## Decision

Go owns durable receiver lifecycle diagnostics:

- unconfigured receiver status storage defaults to
  `.akashic-workspace/agent-runtime/receiver-statuses.json`;
- `AKASHIC_RECEIVER_STATUSES_DSN` and `AKASHIC_RECEIVER_STATUSES_PATH` can
  override the file path;
- `AKASHIC_RECEIVER_STATUSES_DSN=memory` keeps the legacy ephemeral behavior for
  a specific run;
- `AKASHIC_RECEIVER_STATUS_STALE_SECONDS` controls the heartbeat stale window,
  defaulting to 180 seconds and clamped to 30-3600 seconds.

Python receivers publish periodic heartbeats:

- QQ/NapCat channels report `connected` with `reason=heartbeat`;
- Telegram polling reports `connected` with `reason=heartbeat` independently of
  receiver lease renewal, so status can recover after Go restarts;
- clean shutdown reports `stopped`;
- Telegram polling conflicts continue to report `suspended` with
  `reason=getupdates_conflict`.

Go applies stale status at read time. Persisted `connected` or `starting`
receivers older than the stale window are presented as `stopped` with
`reason=heartbeat_stale`, preserving the last status in metadata. This avoids
permanently treating a dead Python receiver as online.

## Boundaries

- Go still does not receive QQ/Telegram messages directly in this slice.
- Python still owns NcatBot/Telegram polling loops and platform SDK callbacks.
- Receiver heartbeat is runtime observability only; it does not send platform
  messages and does not change observe-only reply behavior.
- Receiver leases remain separate from receiver status. Lease persistence and
  reacquire-after-restart are future work if Telegram conflict handling needs a
  stronger guarantee.

## Acceptance

- Receiver status survives `agent-runtime` restart through the file-backed
  repository.
- A stale connected receiver is reported as stopped after the configured TTL.
- QQ channel startup starts a receiver status heartbeat task and shutdown reports
  stopped.
- Telegram polling startup starts a receiver status heartbeat task; conflict
  suspension stops the heartbeat and keeps the suspended status.
- Runtime overview and observe capture diagnostics use the recovered receiver
  status without changing platform side effects.
