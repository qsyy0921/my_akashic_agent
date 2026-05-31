# Observe Capture Activity Inference

## Status

Accepted

## Problem

Observe capture diagnostics previously treated receiver status heartbeat as the
only proof that a QQ observe-only receiver was connected. After restarting only
`agent-runtime`, a still-running Python QQ receiver can continue posting inbox
events before the new heartbeat code is loaded or before the next heartbeat is
observed. In that state diagnostics showed `receiver_not_connected` even while
new group messages, images, and media assets were being captured.

## Decision

Go observe capture diagnostics now distinguish two receiver signals:

- `receiver_status_connected`: a receiver status heartbeat explicitly reports
  `connected`;
- `receiver_activity_recent`: recent observe-only inbox events for the same
  QQ account/group prove that the receive pipeline is active.

The backward-compatible `receiver_connected` field is the effective value:

```text
receiver_connected = receiver_status_connected OR receiver_activity_recent
```

The activity window defaults to 15 minutes inside the Go diagnostics service.
Recent activity only affects read-only diagnostics and blocker classification;
it does not change receiver status state, leases, reply policy, or platform
side effects.

When status heartbeat is missing but recent inbox activity exists, Go reports
`receiver_connection_source=recent_inbox_activity` and does not add the
`receiver_not_connected` blocker. Missing image/file/content coverage remains
reported independently.

## Boundaries

- This slice does not send QQ/Telegram messages.
- It does not replace receiver status heartbeat or receiver leases.
- It does not infer activity from media assets alone; inbox events remain the
  authoritative signal because they prove the Python receive path reached Go.
- File coverage still requires real file assets; image-only groups remain
  `warn` until a file sample is observed.

## Acceptance

- Recent observe-only inbox activity can make `receiver_connected=true` even
  when receiver status heartbeat is absent.
- Stale inbox activity does not hide `receiver_not_connected`.
- Diagnostics expose `receiver_status_connected`,
  `receiver_activity_recent`, and `receiver_connection_source`.
- Runtime overview carries the new observe-capture receiver counters.
- Dashboard normalization preserves the new fields while keeping the existing
  `receiver_connected` contract.
