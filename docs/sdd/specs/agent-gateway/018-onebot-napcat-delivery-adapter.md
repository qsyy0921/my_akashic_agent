# SPEC-018: OneBot/NapCat Delivery Adapter

## Context

`agent-runtime` already owns outbound delivery lifecycle, dispatch planning, and
Telegram HTTP delivery. QQ/NapCat sends are still executed by the Python channel
compatibility layer, which keeps the platform SDK details close to the agent
runtime and prevents outbox delivery from becoming the single infrastructure
boundary.

This slice adds a Go-owned OneBot delivery adapter for NapCat-compatible QQ
accounts. It supports OneBot HTTP action endpoints and OneBot WebSocket action
requests. The current local NapCat containers expose WebSocket servers on
`3001` and `3002`; normal HTTP requests to those ports return `426 Upgrade
Required` because the server expects a WebSocket upgrade.

The runtime also exposes read-only delivery adapter diagnostics and live health
checks so operators can verify channel aliases, transport type, endpoint
presence, access-token presence, and login/auth status without sending QQ or
Telegram messages.

The adapter is deliberately limited to outbound action calls. QR login, inbound
WebSocket observation, group observe-only capture, and media download remain in
the existing Python/NapCat path until those parts receive separate migration
reviews.

## Goals

- Let Go dispatch outbox text, image, and file steps through OneBot HTTP or
  OneBot WebSocket action calls.
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

Multi-account WebSocket deployment matching the current Docker mapping:

```powershell
$env:AKASHIC_ONEBOT_WS_URLS = "qq=ws://127.0.0.1:3001,qq_2365524513=ws://127.0.0.1:3002"
$env:AKASHIC_ONEBOT_ACCESS_TOKENS = "qq=NcatBot,qq_2365524513=NcatBot"
```

Multi-account HTTP deployment:

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

`AKASHIC_ONEBOT_WS_URLS` and `AKASHIC_ONEBOT_WEBSOCKET_URLS` accept the same
`channel=ws://host:port` shape. WebSocket endpoints are preferred over HTTP when
both are configured for the same channel, because current NapCat containers are
already running WebSocket servers.

Read configured delivery adapters without triggering any platform side effect:

```http
GET /v1/delivery-adapters
```

The response includes provider, channel alias, transport, endpoint presence,
redacted endpoint, and whether an access token is configured. It deliberately
does not expose token values and does not perform a live send.

Read sanitized runtime configuration before restarting or sending:

```http
GET /v1/runtime-config
```

The response includes process address/source, bot ids, OneBot expected channel
aliases, configured endpoint aliases, missing aliases, token presence, worker
flags, and `side_effect=none`. For the current two-account deployment the
default expectation is `qq` for the primary local account plus `qq_<bot id>` for
additional bot ids. Operators can override this with
`AKASHIC_ONEBOT_EXPECTED_CHANNELS`. Token/secret environment values are fully
redacted as `redacted` or `channel=redacted`; the endpoint does not expose token
prefixes or suffixes.

Read live adapter health without sending messages:

```http
GET /v1/delivery-adapters/health?timeout_seconds=3
```

OneBot/NapCat health uses `get_login_info` over the configured HTTP or
WebSocket action transport. Telegram health uses `getMe`. The response reports
`healthy`, `reachable`, `authenticated`, `account_id`, `account_name`,
`latency_ms`, and `side_effect=none` per channel alias. This is a pre-send
readiness check, not a substitute for the later live send smoke.

The runtime overview dashboard exposes this through a manual proxy:

```http
GET /api/dashboard/runtime-overview/delivery-adapter-health?timeout_seconds=3
```

The normal overview load path does not call the live health endpoint. The
operator must open the `Delivery Adapters` detail and click the health probe
action.

Read a delivery smoke readiness matrix without creating outbox records or
sending messages:

```http
POST /v1/delivery-smoke/readiness
```

With an empty body, the runtime creates default 104/236-style private
two-account cases from `AKASHIC_BOT_IDS`, including text, synthetic image, and
synthetic file variants. `AKASHIC_DELIVERY_SMOKE_PRIVATE_PAIRS` can override
the private directions using `from>to,from2>to2`.
`AKASHIC_DELIVERY_SMOKE_GROUP_IDS` adds observe-safe group text/image/file
readiness cases. `AKASHIC_DELIVERY_SMOKE_INCLUDE_MEDIA=false` limits generated
defaults to text-only. `AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT` can explicitly map
bot account ids to channel aliases; otherwise `qq_<bot id>` is preferred when
configured, then `qq` for the primary account.

The endpoint returns the same dispatch plan shape plus per-case missing
channels, aggregate totals, blockers, and `side_effect=none`. It only checks
planner output and `DeliveryAdapter.SupportsDeliveryChannel`; live media upload
behavior is still covered by the later explicit send smoke.

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

QQ delivery through Go is env-gated. If no OneBot endpoint is configured,
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
  `gqq:` group inference, configured channel aliases, WebSocket action dispatch,
  and route error mapping.
- `cmd/agent-runtime` tests cover HTTP, WebSocket, multi-endpoint, and
  single-endpoint env parsing.
- Delivery adapter diagnostics cover OneBot WebSocket aliases and Telegram
  channel aliases without leaking token values.
- Runtime config diagnostics cover OneBot expected aliases, missing alias
  blockers, worker flags, and secret redaction without platform side effects.
- Delivery adapter health covers OneBot `get_login_info` and Telegram `getMe`
  without calling message send APIs.
- Runtime overview dashboard exposes a manual health probe action without
  automatically calling live adapter health during panel refresh.
- Delivery smoke readiness covers dual-account private, optional group,
  synthetic image, and synthetic file case planning without sending platform
  messages.
- `DeliveryDispatchStep` exposes `conversation_type` in the query view.
- `config.example.toml` keeps QQ out of Go outbound by default and documents the
  env-gated cutover.
- Python outbox worker only calls Go dispatch for channels listed in
  `integrations.agent_runtime.outbound_channels`, so QQ cutover remains an
  explicit operator decision.
