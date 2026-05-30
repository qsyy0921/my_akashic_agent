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

Persist knowledge/RAG ingestion checkpoints across runtime restarts:

```powershell
$env:AKASHIC_KNOWLEDGE_CHECKPOINTS_DSN = "E:\agent\akashic\.akashic-workspace\runtime\knowledge-checkpoints.json"
```

With this environment variable set, `/v1/knowledge-checkpoints/*` stores
per-source/per-target cursor state in Go. Python RAG workers read this state
before choosing `since_seq` and advance it only after successful non-empty
ingestion.

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

The outbox implementation owns delivery status, attempts, retry, worker lease,
and dead-letter transitions in Go. Telegram can be dispatched directly through
the Go Telegram adapter. QQ/NapCat can be dispatched through the Go OneBot HTTP
adapter when OneBot endpoints are configured; otherwise the Python QQ
compatibility sender remains the fallback path.

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

Enable QQ/NapCat OneBot HTTP delivery adapters explicitly:

```powershell
$env:AKASHIC_ONEBOT_WS_URLS = "qq=ws://127.0.0.1:3001,qq_2365524513=ws://127.0.0.1:3002"
$env:AKASHIC_ONEBOT_ACCESS_TOKENS = "qq=NcatBot,qq_2365524513=NcatBot"
```

The current local NapCat containers expose OneBot WebSocket servers. A plain
HTTP request to those ports returns `426 Upgrade Required`; use WebSocket action
requests unless you explicitly enable NapCat HTTP servers.

OneBot HTTP action endpoints are also supported:

```powershell
$env:AKASHIC_ONEBOT_HTTP_BASE_URLS = "qq_1049511700=http://127.0.0.1:3001,qq_2365524513=http://127.0.0.1:3002"
$env:AKASHIC_ONEBOT_ACCESS_TOKENS = "qq_1049511700=NcatBot,qq_2365524513=NcatBot"
```

For a single endpoint shared by multiple channel aliases:

```powershell
$env:AKASHIC_ONEBOT_HTTP_BASE_URL = "http://127.0.0.1:3001"
$env:AKASHIC_ONEBOT_CHANNELS = "qq,qq_1049511700"
$env:AKASHIC_ONEBOT_ACCESS_TOKEN = "NcatBot"
```

Keep QQ aliases out of `integrations.agent_runtime.outbound_channels` until the
NapCat OneBot endpoint has passed a live send smoke test.

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
local runs allow Akashic workspace upload directories under the repository root,
and can recover the same roots from absolute asset paths inside an Akashic
workspace. Remote platform URLs must be mirrored into a safe root before the
content route will serve them.

Normalized inbound messages and shadow-observed messages also register their
attachments into the media registry automatically. Attachment-provided ids are
preserved as stable `asset_id` values, and duplicate delivery is idempotent.

Create and lease generic agent jobs:

```text
POST /v1/jobs
GET  /v1/jobs?limit=50&type=rag_ingest&status=pending
GET  /v1/jobs/{job_id}
POST /v1/jobs/lease-next
POST /v1/jobs/lease-work
POST /v1/jobs/{job_id}/lease
POST /v1/jobs/{job_id}/renew
POST /v1/jobs/{job_id}/running
POST /v1/jobs/{job_id}/succeeded
POST /v1/jobs/{job_id}/failed
POST /v1/jobs/{job_id}/retry
POST /v1/jobs/{job_id}/cancel
```

The generic job API owns lifecycle, leasing, retry, and dead-letter state. Python
workers still execute image generation, RAG, and memory extraction.
Each lease response includes a `lease_token`. New Python workers pass that token
back to `running`, `succeeded`, and `failed` transitions so Go can reject stale
writebacks after an expired lease is re-leased by another worker. Empty-token
updates remain accepted during the compatibility phase.
Long-running workers also call `POST /v1/jobs/{job_id}/renew` with the current
token while the job is executing. Renew extends the lease without increasing the
attempt count and emits a `renewed` lifecycle event.

Enable strict result writeback fencing after all workers have been upgraded:

```powershell
$env:AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN = "true"
```

In strict mode, `running`, `succeeded`, and `failed` transitions reject empty
`lease_token` values. This is required before generic `agent_job` work can move
from state-store leasing to an external queue acknowledgement protocol.

`POST /v1/jobs/lease-work` is the queue-notification lease entrypoint for future
external consumers. It leases the exact `work_id` from an `agent_job` work
notification and rejects non-`agent_job` work kinds or mismatched aggregate ids.

Persist generic job lifecycle events as a JSONL stream:

```powershell
$env:AKASHIC_AGENT_JOB_EVENTS_DSN = "E:\agent\akashic\.akashic-workspace\runtime\agent-job-events.jsonl"
```

With this environment variable set, every `/v1/jobs` create, lease, running,
succeeded, failed, retry, and cancel transition appends an `AgentJobEvent`.
Inspect the stream through:

```text
GET /v1/job-events?limit=50
GET /v1/job-events?job_id=rag_ingest:qq:3219982:ds1:1
GET /v1/job-events?type=rag_ingest&event=failed
```

Persist outbox delivery lifecycle events as a JSONL stream:

```powershell
$env:AKASHIC_OUTBOX_EVENTS_DSN = "E:\agent\akashic\.akashic-workspace\runtime\outbox-events.jsonl"
```

With this environment variable set, outbound creation, lease, dispatching,
succeeded, failed, and retry transitions append an `OutboxDeliveryEvent`.
Inspect the stream through:

```text
GET /v1/outbox-events?limit=50
GET /v1/outbox-events?delivery_id=outbox-http-1
GET /v1/outbox-events?status=dead_lettered
GET /v1/outbox-events?event=failed
```

Inspect external queue backend migration settings:

```powershell
$env:AKASHIC_QUEUE_BACKEND = "nats_jetstream"
$env:AKASHIC_QUEUE_MODE = "shadow_publish"
$env:AKASHIC_QUEUE_DSN = "nats://127.0.0.1:4222"
$env:AKASHIC_QUEUE_STREAM = "AKASHIC_WORK"
$env:AKASHIC_QUEUE_SUBJECT_PREFIX = "akashic.work"
$env:AKASHIC_QUEUE_CONSUMER_CONCURRENCY = "8"
$env:AKASHIC_QUEUE_MAX_IN_FLIGHT = "64"
```

```text
GET /v1/queue-backend
```

When `nats_jetstream + shadow_publish + AKASHIC_QUEUE_DSN` are configured,
`agent-runtime` publishes work notifications after outbox deliveries and generic
agent jobs are committed to Go state stores. Work discovery and leases still use
Go state stores; NATS is not allowed to execute or lease work in this phase.
The `/v1/queue-backend` response includes `shadow_publish` diagnostics with
publish attempts, success/failure counts, per-subject counts, and sampled
reconciliation deltas against the Go state store and lifecycle event stream.
When `AKASHIC_QUEUE_MODE=dual_read_compare`, the runtime also starts a NATS pull
consumer. It consumes work notifications with a bounded goroutine worker pool,
checks whether each candidate is leaseable in the Go authoritative state store,
acks after recording diagnostics, and does not dispatch platform sends or Python
workers.
When `AKASHIC_QUEUE_MODE=external_lease`, the runtime only exposes a blocked
cutover gate in `/v1/queue-backend`. It reports required checks plus ack, nack,
retry, dead-letter, and rollback policies. Setting the mode alone does not move
work discovery away from Go state-store leasing.
After both explicit cutover flags pass, the first executor consumes only
`outbox` NATS subjects. It leases the Go outbox aggregate by work id, dispatches
through configured Go DeliveryAdapters, then maps success to NATS `ack`,
retryable failure to Go retry + delayed NATS `nack`, terminal failure to NATS
`ack`, and malformed/unsupported work to NATS `term`. Generic `agent_job` work
continues to use Go state-store leasing in this slice. The external lease
diagnostics expose `execution_scope=outbox_delivery_only`, `outbox_delivery` as
the allowed work kind after the gate passes, and `agent_job` as a blocked work
kind until a Python result-ack protocol exists.
NATS JetStream is the preferred first MQ because its subject routing fits
platform/account/job boundaries and its pull consumers can be consumed by a
bounded Go goroutine worker pool. Redis Streams remains a local/simple
deployment alternative; RabbitMQ remains a later option for heavier broker
routing.

Shadow publish subjects:

```text
akashic.work.outbox.{channel_kind}.{account_id}
akashic.work.agent_job.{job_type}
```

Optional dual-read durable consumer name:

```powershell
$env:AKASHIC_QUEUE_DUAL_READ_DURABLE = "AKASHIC_DUAL_READ_COMPARE"
```

External lease cutover flags are intentionally separate:

```powershell
$env:AKASHIC_QUEUE_MODE = "external_lease"
$env:AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER = "true"
$env:AKASHIC_QUEUE_DUAL_READ_SMOKE_PASSED = "true"
$env:AKASHIC_QUEUE_STATE_LEASE_WORKERS_DISABLED = "true"
$env:AKASHIC_QUEUE_EXTERNAL_LEASE_NACK_DELAY_SECONDS = "30"
$env:AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT = "1049511700=qq_1049511700,2365524513=qq_2365524513"
```

The runtime still keeps migration explicit: `AKASHIC_QUEUE_MODE=external_lease`
is not enough by itself. The executor starts only when the provider, DSN,
cutover flag, dual-read smoke flag, and legacy state-store worker shutdown flag
all pass.

Run the local external-lease smoke against a temporary NATS instance:

```powershell
$env:AKASHIC_NATS_SMOKE_DSN = "nats://127.0.0.1:4222"
go test ./smoke -run TestExternalLeaseNATSSmokeOutboxDispositions -count=1 -v
```

The smoke uses a fake DeliveryAdapter and does not send QQ or Telegram messages.

Persist proactive scheduling state across runtime restarts:

```powershell
$env:AKASHIC_PROACTIVE_STATE_DSN = "E:\agent\akashic\.akashic-workspace\runtime\proactive-state.json"
```

This state is deterministic runtime infrastructure: delivery dedupe,
delivery-window counts, context-only send markers, and drift interval markers.
Python still owns prompt selection, LLM decisions, and final proactive content.
When `integrations.agent_runtime.enabled=true`, Python `ProactiveLoop` uses these
routes for scheduling state and keeps SQLite as a compatibility fallback.

```text
POST /v1/proactive/deliveries
GET  /v1/proactive/deliveries?session_key=telegram:100&limit=50
GET  /v1/proactive/deliveries/duplicate?session_key=telegram:100&delivery_key=abc&window_hours=24
GET  /v1/proactive/deliveries/count?session_key=telegram:100&window_hours=24
POST /v1/proactive/context-only
GET  /v1/proactive/context-only/last?session_key=telegram:100
GET  /v1/proactive/context-only/count?session_key=telegram:100&window_hours=24
POST /v1/proactive/drift-runs
GET  /v1/proactive/drift-runs/last?session_key=telegram:100
```

Read and advance knowledge/RAG checkpoints:

```text
GET /v1/knowledge-checkpoints?limit=50&prefix=ragflow:qq:
GET /v1/knowledge-checkpoints/{checkpoint_id}
PUT /v1/knowledge-checkpoints/{checkpoint_id}
```

For QQ group RAGFlow ingest, the checkpoint id convention is:

```text
ragflow:qq:{group_id}:{dataset_id}
```

The checkpoint API owns cursor durability and rejects backwards movement in the
domain layer. RAGFlow upload, parsing, and external dataset behavior remain in
Python.

Inspect memory/RAG worker lifecycle diagnostics:

```text
GET /v1/knowledge-worker-diagnostics?limit=50&stale_after_seconds=900
```

This read-only endpoint combines recent `group_memory_extract` and `rag_ingest`
generic jobs with `memory:` and `ragflow:` checkpoints, status counts, leaseable
counts, and stale lease counts. It is intended for dashboard/ops visibility and
does not execute or mutate jobs.

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
