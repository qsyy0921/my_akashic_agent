# Telegram Delivery Adapter

## Context

`015-delivery-dispatch-plan.md` moved deterministic outbound planning into Go,
but the Python compatibility worker still executes every platform send through
`message_push`. Telegram is the lowest-risk first platform to move behind Go:
it has a stable HTTP Bot API, no local QR login state, and does not require the
Python channel object to hold a WebSocket session.

## Goal

Add a Go-owned Telegram `DeliveryAdapter` that can execute planned outbox
delivery steps for Telegram text, photo, and document sends. Python keeps the
outbox lease/succeeded/failed loop for this slice, but when the delivery route
is Telegram it should ask Go to execute the delivery before falling back to the
existing Python push tool.

## Non-Goals

- Do not migrate QQ/NapCat sends in this slice.
- Do not replace Telegram polling or inbound message handling.
- Do not implement Telegram live edit, streaming preview, Markdown entity
  conversion, or chat actions.
- Do not let Go mark the outbox delivery succeeded/failed yet; Python remains
  the compatibility lifecycle worker until the full delivery worker moves.

## API

```http
POST /v1/delivery-dispatch/send
Content-Type: application/json

{
  "event_id": "outbox event id",
  "channel_by_account": {}
}
```

The endpoint loads the outbox delivery, builds the same dispatch plan as
`/v1/delivery-dispatch/plan`, and sends every step through a registered Go
adapter.

Successful response:

```json
{
  "code": "OK",
  "data": {
    "event_id": "outbox event id",
    "step_count": 1,
    "results": [
      {
        "step_index": 1,
        "kind": "text",
        "channel": "telegram",
        "chat_id": "8655199155",
        "status": "sent",
        "provider_message_id": "123"
      }
    ]
  }
}
```

If no Go adapter is registered for the route, return HTTP `501` with
`error_kind=sender_unavailable`. Python treats that as a fallback signal rather
than a delivery failure.

## Configuration

- `TELEGRAM_BOT_TOKEN` or `AKASHIC_TELEGRAM_BOT_TOKEN`: enables the adapter.
- `AKASHIC_TELEGRAM_API_BASE_URL`: defaults to `https://api.telegram.org`.
- `AKASHIC_TELEGRAM_CHANNELS`: comma-separated channel aliases supported by this
  adapter, default `telegram`.

## Rules

1. Go only executes channels registered with the Telegram adapter.
2. Text steps call `sendMessage`.
3. Image steps call `sendPhoto`; local files use multipart upload and remote
   URLs/file ids are passed as form fields.
4. File steps call `sendDocument`; local files use multipart upload and remote
   URLs/file ids are passed as form fields.
5. Telegram API timeouts map to `platform_timeout`.
6. Telegram route problems such as `chat not found` or `bot was blocked` map to
   `route_error`.
7. Other non-OK Telegram responses map to `platform_error`.
8. Unsupported or malformed steps map to `validation_error` or
   `unsupported_media`.

## Layering

- `domain/model`: delivery result value objects.
- `app/port/out`: `DeliveryAdapter` outbound port.
- `app/service`: `DeliveryDispatchService` orchestrates planning and adapter
  execution.
- `infrastructure/telegramdelivery`: Telegram Bot API adapter.
- `trigger/http`: endpoint serialization only.
- `integrations`: Python worker prefers Go execution for Telegram, then falls
  back to the plan/push path when Go sender is unavailable.

## Acceptance

- Go adapter tests cover `sendMessage`, local `sendPhoto` multipart upload, API
  route errors, and adapter-unavailable fallback semantics.
- Go HTTP tests cover `/v1/delivery-dispatch/send`.
- Python client tests cover runtime dispatch success, unavailable fallback, and
  error-kind propagation.
- Python worker tests prove Telegram uses Go send first while QQ still uses the
  existing plan/push path.
