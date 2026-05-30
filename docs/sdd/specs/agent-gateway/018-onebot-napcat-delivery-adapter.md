# SPEC-018: OneBot/NapCat Delivery Adapter

## Context

`agent-runtime` already owns outbound delivery lifecycle, dispatch planning, and
Telegram HTTP delivery. QQ/NapCat sends are still executed by the Python channel
compatibility layer, which keeps the platform SDK details close to the agent
runtime and prevents outbox delivery from becoming the single infrastructure
boundary.

This slice adds a Go-owned OneBot HTTP delivery adapter for NapCat-compatible QQ
accounts. It is deliberately limited to outbound HTTP sends. QR login, inbound
WebSocket observation, group observe-only capture, and media download remain in
the existing Python/NapCat path until those parts receive separate migration
reviews.

## Goals

- Let Go dispatch outbox text, image, and file steps through OneBot HTTP.
- Support multiple QQ account routes, for example `qq_1049511700` and
  `qq_2365524513`.
- Preserve the DDD/hexagonal boundary: app service depends on the
  `DeliveryAdapter` port, while OneBot/NapCat HTTP details live only in
  `infrastructure/onebotdelivery`.
- Keep QQ delivery disabled by default unless OneBot HTTP endpoints are
  explicitly configured.
- Preserve existing Python fallback and observe-only behavior.

## Non-Goals

- Do not migrate inbound QQ observation or QR login to Go in this slice.
- Do not enable QQ channels in `integrations.agent_runtime.outbound_channels` by
  default.
- Do not change the bot-to-bot protocol or recent-send loop guard policy.
- Do not introduce an external MQ backend yet.

## Configuration Contract

Multi-account deployment:

```powershell
$env:AKASHIC_ONEBOT_HTTP_BASE_URLS = "qq_1049511700=http://127.0.0.1:3001,qq_2365524513=http://127.0.0.1:3002"
$env:AKASHIC_ONEBOT_ACCESS_TOKENS = "qq_1049511700=NcatBot,qq_2365524513=NcatBot"
```

Single endpoint deployment:

```powershell
$env:AKASHIC_ONEBOT_HTTP_BASE_URL = "http://127.0.0.1:3001"
$env:AKASHIC_ONEBOT_CHANNELS = "qq,qq_1049511700"
$env:AKASHIC_ONEBOT_ACCESS_TOKEN = "NcatBot"
```

`AKASHIC_ONEBOT_HTTP_BASE_URLS` takes the form
`channel=http://host:port,channel2=http://host2:port2`. Per-channel tokens from
`AKASHIC_ONEBOT_ACCESS_TOKENS` override the shared
`AKASHIC_ONEBOT_ACCESS_TOKEN`.

## Routing Rules

The dispatch planner maps an outbox delivery to platform steps:

- `channel` comes from `ChannelByAccount[account_id]` when present, otherwise
  from `channel.kind`.
- `chat_id` comes from `conversation_id`.
- `conversation_type` is carried into each dispatch step.

The OneBot adapter resolves targets as follows:

- `chat_id` with prefix `gqq:` is a group target and the prefix is stripped.
- `conversation_type=group` sends through `send_group_msg` or
  `upload_group_file`.
- `conversation_type=private` or empty sends through `send_private_msg` or
  `upload_private_file`.

## Media Rules

- Text steps call `send_private_msg` or `send_group_msg`.
- Image steps call the same message API with CQ/OneBot message segments:
  optional caption text followed by an image segment.
- File steps first send optional caption text, then call `upload_private_file`
  or `upload_group_file`.
- `http://`, `https://`, and `base64://` media values pass through unchanged.
- Local files are read and converted to `base64://...` before calling OneBot.
- Missing local media is classified as `unsupported_media`.

## Error Mapping

The adapter normalizes failures into existing delivery error kinds:

- transport timeout: `platform_timeout`;
- unsupported or missing file/image: `unsupported_media`;
- missing user/group route: `route_error`;
- unavailable endpoint or unconfigured channel: `sender_unavailable`;
- other OneBot failures: `platform_error`.

The outbox domain keeps existing retry policy. Deterministic route, validation,
and unsupported media failures can dead-letter without retry churn.

## Safety

QQ delivery through Go is env-gated. If no OneBot HTTP endpoint is configured,
`agent-runtime` starts without the adapter and existing Python QQ direct send
fallback remains authoritative.

For two-bot interaction, this slice relies on existing controls:

- visible bot protocol tag for intentional bot-to-bot messages;
- Go send ledger for recent-send and echo-loop detection;
- outbox event ids for idempotent delivery state;
- Python compatibility guards while QQ direct fallback remains active.

## Acceptance Criteria

- `go test ./...` passes.
- OneBot adapter unit tests cover private text, group image, private file,
  `gqq:` group inference, configured channel aliases, and route error mapping.
- `cmd/agent-runtime` tests cover multi-endpoint and single-endpoint env parsing.
- `DeliveryDispatchStep` exposes `conversation_type` in the query view.
- `config.example.toml` keeps QQ out of Go outbound by default and documents the
  env-gated cutover.
