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
while deterministic control-plane state defaults to file-backed persistence under
`.akashic-workspace/agent-runtime`. This covers observe targets, receiver
statuses, inbound dedupe, inbox, media assets, send ledger, outbox, generic
jobs, knowledge checkpoints, and proactive state. Individual stores can still
be overridden by their `AKASHIC_*_DSN` or `AKASHIC_*_PATH` variables, and
`memory` remains the explicit opt-out for ephemeral development runs.

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

Default runtime state directory:

```powershell
$env:AKASHIC_RUNTIME_STATE_DIR = "E:\agent\akashic\.akashic-workspace\agent-runtime"
```

If this variable is omitted, `agent-runtime` discovers the Akashic repo root
from the current working directory or executable path and uses the same default
directory. Set it to `memory` to make every unconfigured state store ephemeral.
Per-store `AKASHIC_*_DSN` or `AKASHIC_*_PATH` values still take precedence.

Override observe target persistence:

```powershell
$env:AKASHIC_OBSERVE_TARGETS_DSN = "E:\agent\akashic\.akashic-workspace\runtime\observe-targets.json"
```

Observe targets synced from Python config are file-backed so runtime overview
and observe capture diagnostics can recover after restarting only
`agent-runtime`.

Override receiver status persistence:

```powershell
$env:AKASHIC_RECEIVER_STATUSES_DSN = "E:\agent\akashic\.akashic-workspace\runtime\receiver-statuses.json"
$env:AKASHIC_RECEIVER_LEASES_DSN = "E:\agent\akashic\.akashic-workspace\runtime\receiver-leases.json"
$env:AKASHIC_RECEIVER_STATUS_STALE_SECONDS = "180"
```

Receiver statuses are file-backed so QQ/Telegram connectivity diagnostics can
recover after restarting only `agent-runtime`. Python receivers send periodic
heartbeats; stale `connected` or `starting` heartbeats are shown as `stopped`
after the configured stale window. Receiver leases are file-backed separately
so Telegram polling can continue renewing the same lease token after a short Go
runtime restart; if the lease was lost or expired, Python attempts a guarded
reacquire before suspending polling.

Override agent job persistence:

```powershell
$env:AKASHIC_AGENT_JOBS_DSN = "E:\agent\akashic\.akashic-workspace\runtime\agent-jobs.json"
```

`/v1/jobs` and related lifecycle endpoints use the file-backed `AgentJob` store
so pending/running/failed work remains recoverable after restart.

Override media asset metadata persistence:

```powershell
$env:AKASHIC_MEDIA_ASSETS_DSN = "E:\agent\akashic\.akashic-workspace\runtime\media-assets.json"
```

`/v1/media-assets` and automatic attachment registration use the file-backed
media registry. Use `memory` only for development runs where restart recovery is
not required.

Override outbound send ledger persistence:

```powershell
$env:AKASHIC_SEND_LEDGER_DSN = "E:\agent\akashic\.akashic-workspace\runtime\send-ledger.json"
```

Outbound sends and inbound echo checks share the same file-backed ledger. Use
`memory` only for development runs where recent echo detection does not need
restart recovery.

Override inbound platform message dedupe persistence:

```powershell
$env:AKASHIC_INBOUND_DEDUPE_DSN = "E:\agent\akashic\.akashic-workspace\runtime\inbound-dedupe.json"
```

`/v1/inbound-dedupe/check` stores scoped platform message ids with TTL so
Telegram duplicate deliveries can be dropped across receiver restarts. Python
still keeps a local process dedupe guard and falls back to it if Go is down.

Override raw inbound/observed message persistence:

```powershell
$env:AKASHIC_INBOX_DSN = "E:\agent\akashic\.akashic-workspace\runtime\inbox.json"
```

`/v1/inbound` and `/v1/shadow/inbound` record normalized `InboxEvent` rows in a
file-backed raw message store. Use `memory` only for development runs where
replay and group-memory source recovery are not required.

Override knowledge/RAG ingestion checkpoint persistence:

```powershell
$env:AKASHIC_KNOWLEDGE_CHECKPOINTS_DSN = "E:\agent\akashic\.akashic-workspace\runtime\knowledge-checkpoints.json"
```

`/v1/knowledge-checkpoints/*` stores per-source/per-target cursor state in Go.
Python RAG workers read this state before choosing `since_seq` and advance it
only after successful non-empty ingestion.

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

Inspect Go-owned inbox collection metrics:

```text
GET /v1/inbox-metrics?limit=200
GET /v1/inbox-metrics?channel_kind=qq&conversation_id=27234224&conversation_type=group&observe_only=true&limit=200
```

The metrics response summarizes the bounded raw inbox sample by channel kind,
conversation, decision action, sender kind, observe-only totals, attachment
capture, unique senders, and latest `metadata.seq` cursor per conversation. It
is read-only and does not publish agent inbound work or send platform replies.

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

Inspect configured delivery adapters without sending any platform message:

```text
GET /v1/delivery-adapters
```

The response shows provider, channel alias, transport, endpoint presence,
redacted endpoint, and access-token presence. It does not expose token values.

Inspect the current sanitized runtime configuration:

```text
GET /v1/runtime-config
```

The response shows the process address/source, bot ids, selected
OneBot/Telegram env presence, expected OneBot aliases, missing aliases,
worker/cutover flags, and `side_effect=none`. Tokens, DSNs, and URLs are
redacted. For the local two-account setup, the default expected OneBot aliases
are `qq` for the primary account plus `qq_<bot id>` for additional bot ids; set
`AKASHIC_ONEBOT_EXPECTED_CHANNELS` to override that readiness check.

Probe configured delivery adapter health without sending any platform message:

```text
GET /v1/delivery-adapters/health?timeout_seconds=3
```

OneBot/NapCat probes call `get_login_info` over the configured HTTP or
WebSocket action transport. Telegram probes call `getMe`. The response reports
`healthy`, `reachable`, `authenticated`, account id/name, latency, and
`side_effect=none` for each channel alias. Use this before live QQ/NapCat send
smoke to confirm that both account endpoints are reachable and authenticated.

The browser dashboard exposes the same health probe from the `Runtime Overview`
panel. Open the `Delivery Adapters` detail and use `Probe Health`; normal
overview refreshes do not call live OneBot/Telegram health checks automatically.

Check a QQ/NapCat delivery smoke matrix without creating outbox records or
sending platform messages:

```text
POST /v1/delivery-smoke/readiness
```

With an empty body, the runtime builds default private two-account cases from
`AKASHIC_BOT_IDS`, adds text/image/file variants, and maps accounts to
`qq_<bot id>` aliases when those aliases exist. Configure
`AKASHIC_DELIVERY_SMOKE_GROUP_IDS` to include group text/image/file cases, or
send explicit `cases` in the request body. The endpoint only verifies Go
dispatch planning and `DeliveryAdapter` channel availability; response
`side_effect` is always `none`.

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
POST /v1/jobs/recover-expired
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
`POST /v1/jobs` accepts optional `dedupe_key`; if an active pending, leased, or
running job of the same type/key already exists, Go returns that job instead of
creating another job or publishing another queue work notification. The
observe-only knowledge worker uses this for `group_memory_extract` and
`rag_ingest` backpressure.
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

`POST /v1/jobs/recover-expired` is the explicit timeout recovery endpoint for
future external queue consumers. It scans expired `leased` / `running` jobs,
returns retryable jobs to `pending`, moves exhausted jobs to `dead_lettered`,
clears stale lease ownership, and appends a `lease_expired` lifecycle event.

The same recovery use case can run as an optional runtime background job:

```powershell
$env:AKASHIC_AGENT_JOB_RECOVERY_ENABLED = "true"
$env:AKASHIC_AGENT_JOB_RECOVERY_INTERVAL_SECONDS = "300"
$env:AKASHIC_AGENT_JOB_RECOVERY_LIMIT = "50"
$env:AKASHIC_AGENT_JOB_RECOVERY_RUN_ON_START = "true"
```

The runner is disabled by default. When enabled, it only calls the Go
`RecoverExpiredLeases` application use case; it does not execute Python workers
or acknowledge NATS messages directly.

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

Inspect Go-owned generic job metrics:

```text
GET /v1/job-metrics?job_limit=200&event_limit=200
```

The metrics response summarizes the bounded job sample by status and type,
recent lifecycle throughput by event type, current dead-letter totals, and
recent dead-letter samples. It is read-only and does not lease jobs, execute
Python workers, or acknowledge external queue messages.

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

Inspect Go-owned outbox metrics:

```text
GET /v1/outbox-metrics?delivery_limit=200&event_limit=200
```

The metrics response summarizes the bounded delivery sample by status and
channel kind, recent lifecycle throughput by event type, current dead-letter
totals, and recent dead-letter samples. It is read-only and does not lease
deliveries, send platform messages, or acknowledge external queue messages.

Optionally let Go own local outbox delivery dispatch from the state store:

```powershell
$env:AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED = "true"
$env:AKASHIC_OUTBOX_DELIVERY_WORKER_INTERVAL_SECONDS = "2"
$env:AKASHIC_OUTBOX_DELIVERY_WORKER_BATCH_SIZE = "1"
$env:AKASHIC_OUTBOX_DELIVERY_WORKER_ID = "agent-runtime-outbox-worker"
$env:AKASHIC_OUTBOX_DELIVERY_WORKER_LEASE_TTL_SECONDS = "300"
$env:AKASHIC_OUTBOX_DELIVERY_WORKER_RUN_ON_START = "true"
$env:AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT = "1049511700=qq_1049511700,2365524513=qq_2365524513"
```

The worker is disabled by default. When enabled, it leases outbox deliveries
from Go state storage, marks them dispatching, calls Go `DeliveryAdapter`
dispatch, and writes succeeded/failed state back through the outbox application
service. Do not enable it together with a live NATS `external_lease` outbox
consumer, because both are side-effecting delivery executors.

Inspect Go-owned runtime worker diagnostics:

```text
GET /v1/runtime-workers
```

This read-only endpoint reports whether `agent_job_recovery`,
`outbox_delivery_worker`, `nats_shadow_publisher`, `nats_dual_read_compare`,
and `nats_external_lease` are enabled and running, plus their worker id,
interval, lease TTL, batch size, queue concurrency, max-in-flight, execution
scope, and channel/account attributes. It is intended for dashboard/live-smoke
readiness checks and does not start workers or send platform messages.

Inspect Python AI worker liveness reported into Go:

```text
POST /v1/agent-worker-statuses/report
GET  /v1/agent-worker-statuses?stale_after_seconds=180
```

`image_generation`, `knowledge`, `rag_eval`, and compatibility
`outbox_delivery` Python workers report `starting`, `idle`, `running`,
`failed`, and `stopped` states best-effort. Go stores the latest state in
`agent-worker-statuses.json` by default, applies read-time heartbeat stale
detection, and exposes the result in the runtime overview `Agent Workers` card.
This endpoint is diagnostic only; it does not execute jobs, poll platforms, or
send messages.

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
`ack`, and malformed/unsupported work to NATS `term`.

Generic `agent_job` work still executes in Python, but Go now has the safe
result-ack mapping needed by a future queue cutover: pending/running jobs
delayed-`nack` without being executed by Go, terminal jobs `ack`, missing state
`term`, failed jobs go through Go retry then delayed-`nack`, and expired active
leases are recovered before disposition. The live external lease consumer still
keeps `execution_scope=outbox_delivery_only`; `agent_job` remains blocked until a
NATS-level duplicate-delivery smoke, a pending/running/succeeded flow smoke, and
explicit execution-scope expansion are done.

The optional `agent_job` result-ack scope is deliberately separate from the
outbox cutover. Enable it only after both agent-job NATS smokes have passed:

```powershell
$env:AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN = "true"
$env:AKASHIC_QUEUE_AGENT_JOB_DUPLICATE_SMOKE_PASSED = "true"
$env:AKASHIC_QUEUE_AGENT_JOB_FLOW_SMOKE_PASSED = "true"
$env:AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED = "true"
```

When these flags and the base external-lease gates pass, diagnostics report
`execution_scope=outbox_delivery_and_agent_job_result_ack`. Python still
executes the actual model/RAG/memory jobs; Go only acknowledges terminal queue
notifications and delayed-`nack`s work still waiting for Python result
writeback.

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
go test ./smoke -run "TestExternalLeaseNATSSmoke(OutboxDispositions|AgentJobDuplicateTerminalAck|AgentJobPendingRunningSucceededFlow)" -count=1 -v
```

The smoke uses a fake DeliveryAdapter and fake job state; it does not send QQ or
Telegram messages and does not invoke Python workers.

Persist proactive scheduling state across runtime restarts:

```powershell
$env:AKASHIC_PROACTIVE_STATE_DSN = "E:\agent\akashic\.akashic-workspace\runtime\proactive-state.json"
```

This state is deterministic runtime infrastructure: delivery dedupe,
delivery-window counts, source item seen dedupe, rejection cooldowns,
context-only send markers, drift interval markers, and AnyAction daily quota
windows.
Python still owns prompt selection, LLM decisions, and final proactive content.
When `integrations.agent_runtime.enabled=true`, Python `ProactiveLoop` uses these
routes for scheduling state and keeps SQLite as a compatibility fallback.

```text
POST /v1/proactive/deliveries
GET  /v1/proactive/deliveries?session_key=telegram:100&limit=50
GET  /v1/proactive/deliveries/duplicate?session_key=telegram:100&delivery_key=abc&window_hours=24
GET  /v1/proactive/deliveries/count?session_key=telegram:100&window_hours=24
POST /v1/proactive/seen-items
GET  /v1/proactive/seen-items/seen?source_key=mcp%3Anews&item_id=abc&ttl_hours=24
POST /v1/proactive/rejection-cooldowns
GET  /v1/proactive/rejection-cooldowns/cooled?source_key=mcp%3Anews&item_id=abc&ttl_hours=24
POST /v1/proactive/context-only
GET  /v1/proactive/context-only/last?session_key=telegram:100
GET  /v1/proactive/context-only/count?session_key=telegram:100&window_hours=24
POST /v1/proactive/drift-runs
GET  /v1/proactive/drift-runs/last?session_key=telegram:100
GET  /v1/proactive/anyaction/quota?quota_key=default&reset_hour=12&timezone=Asia%2FShanghai
POST /v1/proactive/anyaction/actions
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

Inspect the Go-owned runtime overview aggregate:

```text
GET /v1/runtime-overview?limit=200&event_limit=50&stale_after_seconds=900
```

This read-only endpoint combines delivery adapter diagnostics, queue backend
state, runtime config diagnostics, runtime worker diagnostics, send ledger
metrics, inbox metrics, agent job metrics, outbox metrics, and knowledge worker
diagnostics into the same summary/card shape consumed by the Python dashboard.
It does not send platform messages, lease work, recover jobs, or mutate runtime
state. The Python dashboard prefers this endpoint and falls back to the older
multi-endpoint read path when it is unavailable.

Record and query recent bot sends:

```text
POST /v1/send-ledger/records
GET  /v1/send-ledger/records?from_bot_id=1049511700&conversation_id=2365524513&limit=50
GET  /v1/send-ledger/recent?from_bot_id=1049511700&conversation_id=2365524513&content=hello&window_seconds=60
GET  /v1/send-ledger/private-echo?from_user_id=1049511700&to_bot_id=2365524513&has_image=true&window_seconds=180
GET  /v1/send-ledger/metrics?from_bot_id=1049511700&conversation_id=2365524513&limit=200
```

`POST /v1/send-ledger/records` accepts either `content` or `content_hash`. When
only `content` is provided, Go computes the same normalized content hash used by
the loop guard. Python compatibility senders should call this endpoint after a
successful QQ/Telegram send until platform dispatch is fully cut over to Go.
When `integrations.agent_runtime.enabled=true`, the Python QQ and Telegram
compatibility channels record successful sends here on a best-effort basis.
`/v1/send-ledger/private-echo` is read-only and centralizes private echo
classification for compatibility channels. Empty-text image/file/forward echoes
are checked with the shared markers `[图片]`, `[文件]`, and `[转发消息]`.
`/v1/send-ledger/metrics` is read-only and summarizes bounded send ledger
coverage by bot, conversation, content hash, and repeated hash risk for loop
guard audits.

Check inbound platform message duplicates:

```text
POST /v1/inbound-dedupe/check
GET  /v1/inbound-dedupe/records?scope=telegram:telegram&limit=100
GET  /v1/inbound-dedupe/metrics?scope=telegram:telegram&limit=100
```

`POST /v1/inbound-dedupe/check` accepts `scope`, `message_key`,
`ttl_seconds`, optional `timestamp`, and non-secret `metadata`. It returns
`duplicate`, `seen_count`, expiry timestamps, and
`side_effect=runtime_state_only`. The endpoint only mutates dedupe state; it
does not publish inbound work or send platform messages.
`GET /v1/inbound-dedupe/metrics` is read-only, reports
`side_effect=none`, and summarizes duplicate suppression by scope for runtime
overview diagnostics.

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
