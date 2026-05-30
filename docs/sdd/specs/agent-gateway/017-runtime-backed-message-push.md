# Runtime-backed Message Push

## Context

Telegram delivery sending now exists in Go, but normal replies and proactive
messages still usually call Python `message_push` channel senders directly.
That leaves the common outbound path outside the Go outbox lifecycle unless a
caller already creates an outbox delivery manually.

## Goal

Let normal Python `message_push` and outbound port calls enqueue supported
messages into Go `/v1/outbound` when `agent_runtime.outbox_worker_enabled=true`.
The Go outbox worker then leases and dispatches them through Go-owned delivery
adapters when available.

## Non-Goals

- Do not force every platform through Go in this slice.
- Do not remove direct Python channel senders; they remain the safety fallback.
- Do not make the outbox worker call the runtime-backed push path, because that
  would enqueue a delivery while dispatching a delivery.
- Do not change Telegram inbound polling, live preview, or streaming behavior.

## Design

`MessagePushTool` gains an optional runtime enqueue hook:

```text
agent tool call
  -> MessagePushTool.execute(...)
  -> if channel is runtime-enabled and outbox worker is enabled:
       AgentRuntimeOutboundEnqueuer.enqueue(...)
       POST /v1/outbound
       return a queued/sent-compatible success string
     else:
       existing direct sender path
```

The outbox compatibility worker calls `execute_direct(...)` on the same tool, so
fallback dispatch cannot recurse back into `/v1/outbound`.

## Runtime Channels

This slice defaults runtime-backed push to Telegram only. QQ/NapCat remains on
the direct path until a Go QQ adapter is designed and implemented.

Configuration:

```toml
[integrations.agent_runtime]
enabled = true
outbox_worker_enabled = true
outbound_channels = ["telegram"]
```

## Payload Mapping

- `channel` maps to `channel.kind`.
- `chat_id` maps to `channel.conversation_id`.
- `account_id` is resolved from configured channel metadata; if unavailable it
  falls back to the channel name.
- `message` maps to `content`.
- `image` maps to an `image` attachment.
- `file` maps to a `file` attachment.
- Metadata includes `source=message_push` and `runtime_outbound=true`.

## Acceptance

- `MessagePushTool` enqueues runtime-enabled Telegram sends and falls back to
  direct sender if runtime enqueue fails.
- `AgentGatewayOutboxWorker` uses direct sends for Python fallback dispatch.
- `AgentGatewayClient` covers `/v1/outbound`.
- Bootstrap wiring enables runtime-backed push only when both runtime and
  outbox worker are enabled.
- Chinese TODO and review docs are updated in the same slice.
