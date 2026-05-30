# Akashic Agent Runtime

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

The implementation keeps message event fanout in-memory to keep startup simple,
while durable control-plane state can be enabled independently for `AgentJob`,
media asset, send ledger, inbox, and outbox delivery records.

## Run Locally

```powershell
$goRoot = "$env:USERPROFILE\.codex\tools\go1.26.3"
$env:PATH = "$goRoot\bin;$env:PATH"
cd E:\agent\akashic\services\agent-runtime
$env:AKASHIC_BOT_IDS = "1049511700,2365524513"
go run ./cmd/agent-runtime
```

Default address:

```text
:8780
```

Override with:

```powershell
$env:AKASHIC_RUNTIME_ADDR = ":8780"   # 推荐
# 或兼容旧命名（仍可用）
$env:AKASHIC_GATEWAY_ADDR = ":8780"
```

Persist shadow audit events across runtime restarts:

```powershell
$env:AKASHIC_SHADOW_AUDIT_PATH = "E:\agent\akashic\.akashic-workspace\shadow\runtime-audit.jsonl"
```

When this variable is set, `/v1/shadow/observed` reads recent events from the
JSONL audit file instead of the in-memory development store.

Persist agent jobs across runtime restarts:

```powershell
$env:AKASHIC_AGENT_JOBS_DSN = "E:\agent\akashic\.akashic-workspace\runtime\agent-jobs.json"
```

With this environment variable set, `/v1/jobs` and related lifecycle endpoints use
the file-backed `AgentJob` store so pending/running/failed work remains
recoverable after restart.

Persist media asset metadata across runtime restarts:

```powershell
$env:AKASHIC_MEDIA_ASSETS_DSN = "E:\agent\akashic\.akashic-workspace\runtime\media-assets.json"
```

With this environment variable set, `/v1/media-assets` and automatic attachment
registration use the file-backed media registry. Use `memory` only for
development runs where restart recovery is not required.

Persist outbound send ledger records across runtime restarts:

```powershell
$env:AKASHIC_SEND_LEDGER_DSN = "E:\agent\akashic\.akashic-workspace\runtime\send-ledger.json"
```

With this environment variable set, outbound sends and inbound echo checks share
the same file-backed ledger. Use `memory` only for development runs where recent
echo detection does not need restart recovery.

Persist raw inbound/observed message events across runtime restarts:

```powershell
$env:AKASHIC_INBOX_DSN = "E:\agent\akashic\.akashic-workspace\runtime\inbox.json"
```

With this environment variable set, `/v1/inbound` and `/v1/shadow/inbound`
record normalized `InboxEvent` rows in a file-backed raw message store. Use
`memory` only for development runs where replay and group-memory source
recovery are not required.

## HTTP Contracts

Health:

```text
GET /healthz
```

Normalize and route inbound platform messages:

```text
POST /v1/inbound
```

Query raw inbound/observed message events:

```text
GET /v1/inbox?channel_kind=qq&conversation_id=27234224&conversation_type=group&observe_only=true&limit=50
GET /v1/inbox/{event_id}
```

The inbox is the Go-owned raw message log for QQ/TG observations. It preserves
route, sender, provenance, decision, attachment metadata, and observe-only
state. Duplicate `event_id` writes are idempotent.

Publish outbound messages:

```text
POST /v1/outbound
```

Query and update outbound delivery state:

```text
GET  /v1/outbox?limit=50
GET  /v1/outbox/{event_id}
POST /v1/outbox/lease-next
POST /v1/outbox/{event_id}/dispatching
POST /v1/outbox/{event_id}/succeeded
POST /v1/outbox/{event_id}/failed
POST /v1/outbox/{event_id}/retry
```

The current outbox implementation is a control-plane migration slice. It owns
delivery status, attempts, retry, worker lease, and dead-letter transitions in
Go, while the actual QQ/Telegram SDK send path remains on the Python
compatibility layer until the platform adapter cutover is reviewed.

`POST /v1/outbox/lease-next` accepts:

```json
{
  "worker_id": "qq-dispatcher",
  "ttl_seconds": 300
}
```

It returns the next queued delivery, or an expired `dispatching` delivery, as
`dispatching` with `lease_owner` and `lease_expires_at` set.

Failure requests accept structured diagnostics:

```json
{
  "error_kind": "platform_timeout",
  "error_message": "platform timeout"
}
```

Known `error_kind` values are `unknown`, `platform_error`,
`platform_timeout`, `route_error`, `unsupported_media`,
`sender_unavailable`, and `validation_error`. Omitted or unrecognized values are
normalized to `unknown` by the domain layer. `retry`, `dispatching`, and
`succeeded` clear previous failure details.

Retryability is also owned by the domain layer. `route_error`,
`unsupported_media`, and `validation_error` are deterministic failures and move
directly to `dead_lettered`; `unknown`, `platform_error`, `platform_timeout`,
and `sender_unavailable` remain retryable until max attempts are exhausted.

Register and query media/file metadata:

```text
POST /v1/media-assets
GET  /v1/media-assets?limit=50
GET  /v1/media-assets/{asset_id}
GET  /v1/media-assets/{asset_id}/content
```

The first media registry slice is metadata-only. The `/content` route returns
bytes only for registered local files under configured safe roots. Configure
roots with `AKASHIC_MEDIA_ASSET_ROOTS` as a comma-separated list. If omitted,
local runs allow Akashic workspace upload directories under the repository root.
Remote platform URLs must be mirrored into a safe root before the content route
will serve them.

Normalized inbound messages and shadow-observed messages also register their
attachments into the media registry automatically. Attachment-provided ids are
preserved as stable `asset_id` values, and duplicate delivery is idempotent.

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

Record and query recent bot sends:

```text
POST /v1/send-ledger/records
GET  /v1/send-ledger/records?from_bot_id=1049511700&conversation_id=2365524513&limit=50
GET  /v1/send-ledger/recent?from_bot_id=1049511700&conversation_id=2365524513&content=hello&window_seconds=60
```

`POST /v1/send-ledger/records` accepts either `content` or `content_hash`. When
only `content` is provided, Go computes the same normalized content hash used by
the loop guard. Python compatibility senders should call this endpoint after a
successful QQ/Telegram send until platform dispatch is fully cut over to Go.
When `integrations.agent_runtime.enabled=true`, the Python QQ and Telegram
compatibility channels record successful sends here on a best-effort basis.

Persist outbound delivery state across runtime restarts:

```powershell
$env:AKASHIC_OUTBOX_DSN = "E:\agent\akashic\.akashic-workspace\runtime\outbox.json"
```

With this environment variable set, `/v1/outbound` and `/v1/outbox/*` use the
file-backed outbox store. Queued, dispatching, failed, retried, succeeded, and
dead-letter state remains inspectable after `agent-runtime` restarts. Use
`memory` only for development runs where delivery recovery is not required.

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
cd E:\agent\akashic\services\agent-runtime
gofmt -w api app cmd domain infrastructure trigger types
go test ./...
go build ./cmd/agent-runtime
```
