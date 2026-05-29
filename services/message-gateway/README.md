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

Persist shadow audit events across gateway restarts:

```powershell
$env:AKASHIC_SHADOW_AUDIT_PATH = "E:\agent\akashic\.akashic-workspace\shadow\gateway-audit.jsonl"
```

When this variable is set, `/v1/shadow/observed` reads recent events from the
JSONL audit file instead of the in-memory development store.

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

Query and update outbound delivery state:

```text
GET  /v1/outbox?limit=50
GET  /v1/outbox/{event_id}
POST /v1/outbox/{event_id}/dispatching
POST /v1/outbox/{event_id}/succeeded
POST /v1/outbox/{event_id}/failed
POST /v1/outbox/{event_id}/retry
```

The current outbox implementation is a control-plane migration slice. It owns
delivery status, attempts, retry, and dead-letter transitions in Go, while the
actual QQ/Telegram SDK send path remains on the Python compatibility layer until
the platform adapter cutover is reviewed.

Register and query media/file metadata:

```text
POST /v1/media-assets
GET  /v1/media-assets?limit=50
GET  /v1/media-assets/{asset_id}
GET  /v1/media-assets/{asset_id}/content
```

The first media registry slice is metadata-only. The `/content` route returns
`501 Not Implemented` until workspace path validation and controlled asset
access are reviewed and tested.

Create and lease generic agent jobs:

```text
POST /v1/jobs
GET  /v1/jobs?limit=50&type=rag_ingest&status=pending
GET  /v1/jobs/{job_id}
POST /v1/jobs/lease-next
POST /v1/jobs/{job_id}/lease
POST /v1/jobs/{job_id}/running
POST /v1/jobs/{job_id}/succeeded
POST /v1/jobs/{job_id}/failed
POST /v1/jobs/{job_id}/retry
POST /v1/jobs/{job_id}/cancel
```

The generic job API owns lifecycle, leasing, retry, and dead-letter state. Python
workers still execute image generation, RAG, and memory extraction.

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
