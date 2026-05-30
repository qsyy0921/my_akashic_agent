# Delivery Dispatch Plan

## Context

Outbound delivery already has a Go-owned control plane: outbox records,
leases, retryability, failed/dead-letter transitions, and dashboard diagnostics.
The remaining Python compatibility worker still contains deterministic
delivery-adapter logic:

- map bot account id to configured channel name;
- validate route and chat id;
- split image/file/text payloads into send steps;
- normalize local `file://` URLs into platform-tool file paths.

Those decisions are infrastructure concerns and should move behind Go
`agent-runtime` without immediately migrating QQ/Telegram SDK calls.

## Goal

Introduce a Go-owned `DeliveryPlanner` that turns one outbox delivery into a
stable dispatch plan. Python remains a platform sender for this slice, but it
must execute the Go plan when the runtime is available.

## Non-Goals

- Do not replace `message_push` or NcatBot/Telegram SDK sends in this slice.
- Do not add a new external queue backend.
- Do not change observe-only group ingestion or image-generation job behavior.
- Do not bypass existing outbox lease/succeeded/failed transitions.

## API

```http
POST /v1/delivery-dispatch/plan
Content-Type: application/json

{
  "event_id": "outbox event id",
  "channel_by_account": {
    "2365524513": "qq_2365524513"
  }
}
```

Response:

```json
{
  "code": "OK",
  "data": {
    "event_id": "outbox event id",
    "channel": "qq_2365524513",
    "chat_id": "1049511700",
    "step_count": 2,
    "steps": [
      {
        "step_index": 1,
        "kind": "image",
        "channel": "qq_2365524513",
        "chat_id": "1049511700",
        "message": "caption",
        "image": "E:/agent/akashic/.tmp/a.png"
      },
      {
        "step_index": 2,
        "kind": "file",
        "channel": "qq_2365524513",
        "chat_id": "1049511700",
        "file": "E:/agent/akashic/.tmp/a.pdf"
      }
    ]
  }
}
```

Planning errors return `INVALID_ARGUMENT` with an `error_kind` compatible with
outbox failure classification:

```json
{
  "code": "INVALID_ARGUMENT",
  "message": "outbox delivery missing channel kind",
  "data": {"error_kind": "route_error"}
}
```

## Rules

1. Account mapping has priority over raw platform kind. If account
   `2365524513` maps to `qq_2365524513`, all steps use that channel name.
2. `conversation_id` becomes the `chat_id` passed to the platform sender.
3. Image attachments are sent first; the original content is attached only to
   the first send step.
4. Non-image attachments are treated as file sends.
5. If there are no attachments, one text step is emitted.
6. Local `file://` URLs are normalized to filesystem paths so Python
   `message_push` can pass them to channel adapters.
7. Route errors are non-retryable and are surfaced as `route_error`; malformed
   delivery records are surfaced as `validation_error`.

## Layering

- `domain/service`: pure `DeliveryPlanner`.
- `app/service`: loads the outbox delivery and maps domain plan to query view.
- `trigger/http`: exposes the plan endpoint and serializes error kind.
- `integrations`: Python compatibility worker asks Go for a plan, then executes
  platform sends through the existing tool.

## Acceptance

- Go unit tests cover account mapping, text/image/file split, and route errors.
- Go HTTP tests cover the dispatch plan endpoint.
- Python client tests cover plan request and Go error-kind propagation.
- Python worker tests prove runtime plan execution is preferred and errors are
  written back to outbox with the Go-provided kind.
