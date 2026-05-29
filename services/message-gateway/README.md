# Akashic Message Gateway

This Go service is the infrastructure boundary for platform messaging. It keeps
QQ, Telegram, queue routing, bot identity, loop protection, media events, and
audit events out of the Python Agent runtime.

## Architecture

```text
api             external DTOs and contracts
app             use cases, commands, ports, and orchestration
domain          pure message and loop-guard business rules
infrastructure  outbound adapters such as queue, storage, and platform clients
trigger         inbound adapters such as HTTP, MQ listeners, and jobs
types           shared non-business primitives
```

Dependency direction:

```text
trigger        -> api, app
app            -> domain, types
infrastructure -> app, domain, types
api            -> types
domain         -> types only when unavoidable
types          -> none
```

The first implementation intentionally uses an in-memory event bus so the
domain and app boundaries can be tested before NATS JetStream, NapCat adapters,
and persistent audit storage are introduced.

## Run Locally

```powershell
$goRoot = "$env:USERPROFILE\.codex\tools\go1.26.3"
$env:PATH = "$goRoot\bin;$env:PATH"
cd E:\agent\akashic\services\message-gateway
$env:AKASHIC_BOT_IDS = "1049511700,2365524513"
go run ./cmd/message-gateway
```

Default address:

```text
:8780
```

Override with:

```powershell
$env:AKASHIC_GATEWAY_ADDR = ":8780"
```

## HTTP Contracts

Health:

```text
GET /healthz
```

Normalize and route inbound platform messages:

```text
POST /v1/inbound
```

Publish outbound messages:

```text
POST /v1/outbound
```

For controlled bot-to-bot interaction, set `with_bot_protocol=true` on outbound
requests. The app layer prepends a visible protocol tag:

```text
[[akashic:bot from=1049511700 nonce=<nonce> hop=<n>]]
```

Inbound peer-bot messages without this tag are observed and audited, but are not
forwarded to the Agent for reply.

## Verify

```powershell
$goRoot = "$env:USERPROFILE\.codex\tools\go1.26.3"
$env:PATH = "$goRoot\bin;$env:PATH"
cd E:\agent\akashic\services\message-gateway
gofmt -w api app cmd domain infrastructure trigger types
go test ./...
go build ./cmd/message-gateway
```
