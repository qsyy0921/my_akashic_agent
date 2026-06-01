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

Check whether outbound sending is ready to cut over to Go:

```text
POST /v1/outbound-cutover/readiness
```

This read-only gate combines sanitized OneBot runtime config, delivery smoke
readiness, queue backend execution owner, and runtime worker diagnostics. It is
ready only when expected OneBot aliases are configured, the smoke matrix can be
planned to available adapters, and either the Go local outbox worker or NATS
external lease can execute `outbox_delivery`. It does not send platform
messages, enqueue outbox records, or mutate worker flags.

Build a read-only outbound cutover plan:

```text
POST /v1/outbound-cutover/plan
```

The plan endpoint accepts the same smoke matrix fields as readiness plus
`desired_execution_owner` (`auto`, `go_local_outbox_worker`, or
`nats_external_lease`). It embeds the readiness result and returns required
checks, enable steps, verification endpoints, rollback steps, blockers, and
`side_effect=none`. The endpoint recommends environment keys but never writes
them, starts workers, enqueues deliveries, or sends QQ/Telegram messages.

Register and query media/file metadata:

```text
POST /v1/media-assets
GET  /v1/media-assets?limit=50
GET  /v1/media-assets/content-diagnostics?limit=50
GET  /v1/media-assets/content-diagnostics?asset_id=asset%3Aqq%3A...
GET  /v1/media-assets/content-access-plan?asset_id=asset%3Aqq%3A...
GET  /v1/media-assets/content-recovery-plan?asset_id=asset%3Aqq%3A...
GET  /v1/media-assets/content-recovery/preflight?asset_id=asset%3Aqq%3A...&operator_id=qsyy&approval_id=...
POST /v1/media-assets/content-recovery
GET  /v1/media-assets/retention-diagnostics?limit=50
GET  /v1/media-assets/retention-plan?limit=50
GET  /v1/media-assets/retention-cleanup/preflight?target_id=default-observed-group&operator_id=qsyy&approval_id=...
POST /v1/media-assets/retention-cleanup
GET  /v1/media-assets/{asset_id}
GET  /v1/media-assets/{asset_id}/content
```

The media registry is Go-owned metadata. The `/content` route returns bytes only
for registered local files under configured safe roots. Configure roots with
`AKASHIC_MEDIA_ASSET_ROOTS` as a comma-separated list. If omitted, local runs
allow Akashic workspace upload directories under the repository root, and can
recover the same roots from absolute asset paths inside an Akashic workspace.
Remote platform URLs must be mirrored into a safe root before the content route
will serve them.

The content diagnostics endpoint is read-only and classifies each asset as
`ready`, `forbidden`, `unavailable`, `disabled`, or `error`. Each item includes
the stable content route plus single-asset `content_access_plan_endpoint`,
`content_recovery_plan_endpoint`, and `content_recovery_preflight_endpoint`
fields for dashboard drilldown. It does not run
OCR/VLM, parse files, upload to RAG, or change the `/content` access policy. The
content access and recovery plan endpoints are also read-only: they probe
deterministic availability, immediately close any opened file, return
ready/reason/blockers and operational steps, and do not stream content to the
caller.

`POST /v1/media-assets/content-recovery` is the approval-bound executor for
HTTP/HTTPS media redownload into a local cache root. It always runs the
preflight first, supports `dry_run`, writes only under
`AKASHIC_MEDIA_CONTENT_RECOVERY_CACHE_ROOT` (defaulting to an Akashic upload
root when discoverable), updates the Go media registry with `local_path`,
hash/size/mime metadata, and records applied/failed control mutation audit.
It does not use QQ/Telegram private credentials, parse files, run OCR/VLM,
enqueue RAG jobs, or invoke Python AI.

Retention diagnostics and retention plan are read-only: they report cleanup
candidates and operator approval / control mutation audit steps, but do not
delete media metadata, remove local files, enqueue jobs, or call Python
OCR/VLM/file parsing. The retention cleanup preflight validates cleanup
candidates plus an active operator approval and returns `side_effect=none`.
The cleanup executor requires that preflight, records applied/failed control
mutation audit, and deletes only Go media asset metadata. It never deletes local
files and never triggers OCR/VLM/RAG/AI.

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

Optionally let Go own recurring observe-only knowledge job admission:

```powershell
$env:AKASHIC_KNOWLEDGE_JOB_PLANNER_ENABLED = "true"
$env:AKASHIC_KNOWLEDGE_JOB_PLANNER_INTERVAL_SECONDS = "60"
$env:AKASHIC_KNOWLEDGE_JOB_PLANNER_WORKER_ID = "agent-runtime-knowledge-job-planner"
$env:AKASHIC_KNOWLEDGE_JOB_PLANNER_AGENT_ID = "akashic-python-worker"
$env:AKASHIC_KNOWLEDGE_JOB_PLANNER_MAX_ATTEMPTS = "2"
$env:AKASHIC_KNOWLEDGE_JOB_PLANNER_RAG_MAX_MESSAGES = "1000"
$env:AKASHIC_KNOWLEDGE_JOB_PLANNER_RAG_PARSE = "true"
$env:AKASHIC_KNOWLEDGE_JOB_PLANNER_RUN_ON_START = "true"
```

The planner is disabled by default. When enabled, it lists Go-owned observe
targets, creates `group_memory_extract` jobs for enabled observe-only QQ group
targets, and creates `rag_ingest` jobs for configured
`ragflow_dataset_ids`. Existing AgentJob `dedupe_key` admission remains the
backpressure guard, so repeated planning rounds do not create duplicate active
work. Python knowledge workers still lease and execute the jobs; when they see
`/v1/runtime-config.workers.knowledge_job_planner_enabled=true`, they skip
their legacy enqueue loop and act as execution workers only.

Preview the exact planner output before enabling real admission:

```text
GET /v1/knowledge-job-planner/preview
GET /v1/knowledge-job-planner/preview?timestamp=2026-05-31T08:02:00Z&interval_seconds=60
GET /v1/knowledge-job-planner/readiness
```

The preview is read-only (`side_effect="none"`). It returns eligible
observe-only QQ groups, skipped targets with reasons, planned
`group_memory_extract` / `rag_ingest` job ids, routes, payloads, dedupe keys,
and dataset bindings. It does not create AgentJob records.
The readiness endpoint combines the same preview with runtime config,
`knowledge_job_planner` runtime-worker state, and Python `worker_type=knowledge`
heartbeat status. A disabled planner is treated as preflight, not a blocker; an
enabled planner that is not running is blocked.

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
recent lifecycle throughput by event type, current dead-letter totals, recent
dead-letter samples, and job-type pressure for pending / leased / running
backlog. Pressure is read-only: Go reports `pending`, `active`,
`oldest_pending_age_seconds`, and high-pressure reasons by `job_type`, but does
not auto-scale Python workers, reject enqueue, or change retry/lease behavior.
It is read-only and does not lease jobs, execute Python workers, or acknowledge
external queue messages.

Inspect the read-only AgentJob capacity plan:

```text
GET /v1/agent-job-capacity/plan?job_limit=200&event_limit=50&stale_after_seconds=900
```

The capacity plan combines AgentJob pressure with Python AI worker status. It
recommends operational actions such as recovering a missing knowledge worker,
restarting stale workers, inspecting failed workers, or tuning Python worker
concurrency when active workers exist but pressure remains high. It is advisory
only: Go does not start workers, change concurrency, lease/retry jobs,
acknowledge MQ messages, or execute model/RAG/memory/OCR/VLM/image work.

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
totals, recent dead-letter samples, and account-level pressure by
`channel_kind:account_id`. It is read-only and does not lease deliveries, send
platform messages, or acknowledge external queue messages.

Optionally let Go own local outbox delivery dispatch from the state store:

```powershell
$env:AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED = "true"
$env:AKASHIC_OUTBOX_DELIVERY_WORKER_INTERVAL_SECONDS = "2"
$env:AKASHIC_OUTBOX_DELIVERY_WORKER_BATCH_SIZE = "1"
$env:AKASHIC_OUTBOX_DELIVERY_WORKER_ID = "agent-runtime-outbox-worker"
$env:AKASHIC_OUTBOX_DELIVERY_WORKER_LEASE_TTL_SECONDS = "300"
$env:AKASHIC_OUTBOX_DELIVERY_WORKER_RUN_ON_START = "true"
$env:AKASHIC_OUTBOX_DELIVERY_ACCOUNT_MIN_INTERVAL_SECONDS = "3"
$env:AKASHIC_OUTBOX_DELIVERY_ACCOUNT_WINDOW_SECONDS = "60"
$env:AKASHIC_OUTBOX_DELIVERY_ACCOUNT_MAX_PER_WINDOW = "5"
$env:AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT = "1049511700=qq_1049511700,2365524513=qq_2365524513"
```

The worker is disabled by default. When enabled, it leases outbox deliveries
from Go state storage, marks them dispatching, calls Go `DeliveryAdapter`
dispatch, and writes succeeded/failed state back through the outbox application
service. Optional account throttling skips rate-limited account keys before
leasing so blocked deliveries stay queued and other accounts can continue.
Do not enable it together with a live NATS `external_lease` outbox consumer,
because both are side-effecting delivery executors.

Inspect Go-owned runtime worker diagnostics:

```text
GET /v1/runtime-workers
```

This read-only endpoint reports whether `agent_job_recovery`,
`outbox_delivery_worker`, `knowledge_job_planner`, `nats_shadow_publisher`,
`nats_dual_read_compare`, and `nats_external_lease` are enabled and running,
plus their worker id, interval, lease TTL, batch size, queue concurrency,
max-in-flight, execution scope, and channel/account attributes. It is intended
for dashboard/live-smoke readiness checks and does not start workers or send
platform messages.

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
New Python reporters include `instance_id` and `lease_ttl_seconds`; Go rejects
active same-`worker_id` status overwrites from a different instance until the
lease expires or the owning instance reports `stopped`/`failed`.
Long-running Python AI workers also renew this diagnostic worker status lease by
periodically reporting `running` with the current job id while image,
knowledge, RAG eval, or compatibility outbox work is executing. This is separate
from AgentJob lease renewal and only keeps the worker liveness/fencing view
fresh.
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
When the external lease consumer is actually running, `/v1/queue-backend`
also includes `external_lease.diagnostics` with bounded recent execution
samples and disposition / reason / work-kind counters. These diagnostics are
read-only observations of Go's queue ack/nack/term decisions; the authoritative
state remains the outbox, AgentJob, and lifecycle event stores.
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

Check whether `agent_job` result-ack is ready for external lease:

```text
GET /v1/agent-job-external-lease/readiness?stale_after_seconds=900
```

This read-only gate combines `/v1/queue-backend`, strict AgentJob lease-token
runtime config, `AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED`, AgentJob
pressure, and Python worker heartbeat coverage. It is ready only when
`agent_job` is included in the external lease allowed work kinds, execution
owner is `python_ai_worker_with_nats_result_ack`, strict lease tokens are
enabled, and high-pressure job types do not have danger-level Python worker
coverage. It does not lease jobs, acknowledge NATS messages, start Python
workers, or execute model/RAG/memory work.

Plan `agent_job` result-ack cutover without changing runtime state:

```text
GET /v1/agent-job-external-lease/plan?desired_execution_owner=nats_result_ack
```

The plan returns required checks, env hints, verification endpoints and rollback
steps for moving only generic `agent_job` queue acknowledgement into NATS
external lease. Python keeps executing AI jobs; Go only controls AgentJob
lifecycle, result-ack mapping and read-only diagnostics. Rollback removes the
`agent_job` result-ack scope without disabling the outbox external lease path.

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
context-only send markers, drift interval markers, drift skill run state,
drift recent-run summaries, proactive tick audit logs, proactive tick step
logs, and AnyAction daily quota windows.
Python still owns prompt selection, LLM decisions, and final proactive content.
When `integrations.agent_runtime.enabled=true`, Python `ProactiveLoop` uses these
routes for scheduling state and keeps SQLite as a compatibility fallback.
Python `DriftStateStore` also uses the drift routes for `run_count`, `status`,
`next`, recent runs, and note, while still mirroring to workspace JSON files as
a fallback. Skill file scanning, `SKILL.md` parsing, and drift tool execution
remain in Python.
Python also writes tick start/finish/step audit events to Go first, then keeps
the existing SQLite `tick_log` / `tick_step_log` mirror for dashboard
compatibility. The dashboard keeps SQLite as the primary read path and falls
back to Go tick list/detail/steps when SQLite has no matching records.

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
POST /v1/proactive/drift/finish
GET  /v1/proactive/drift/summary?limit=10
GET  /v1/proactive/drift/skills/explore-curiosity
POST /v1/proactive/tick-logs/start
POST /v1/proactive/tick-logs/finish
POST /v1/proactive/tick-steps
GET  /v1/proactive/tick-logs?terminal_action=reply&limit=50&offset=0&sort_by=started_at&sort_order=desc
GET  /v1/proactive/tick-logs/tick-1
GET  /v1/proactive/tick-logs/tick-1/steps
POST /v1/proactive/bg-context/main
GET  /v1/proactive/bg-context/main/last
GET  /v1/proactive/anyaction/quota?quota_key=default&reset_hour=12&timezone=Asia%2FShanghai
POST /v1/proactive/anyaction/actions
POST /v1/proactive/cleanup
```

Persist Python scheduler job snapshots in Go while keeping the Python tick loop
and AI execution unchanged:

```powershell
$env:AKASHIC_SCHEDULER_JOBS_DSN = "E:\agent\akashic\.akashic-workspace\runtime\scheduler-jobs.json"
$env:AKASHIC_SCHEDULER_LEASES_DSN = "E:\agent\akashic\.akashic-workspace\runtime\scheduler-leases.json"
```

When `integrations.agent_runtime.enabled=true`, Python `JobStore` writes the
complete scheduler snapshot to Go and still keeps the local `schedules.json` as a
fallback. Python `SchedulerService` also acquires a Go-owned execution lease
before running a due job, renews it during long execution, and releases it after
the scheduler snapshot has been saved. Tool-driven add/cancel paths use single
job upsert/delete so one Python process does not have to replace the whole
snapshot when registering or cancelling a reminder. Execution completion uses a
lease-fenced complete mutation so the holder that ran the job is the only one
allowed to reschedule or delete it.

```text
GET  /v1/scheduler/jobs
POST /v1/scheduler/jobs/snapshot
POST /v1/scheduler/jobs/upsert
DELETE /v1/scheduler/jobs/{job_id}
POST /v1/scheduler/jobs/{job_id}/complete
GET  /v1/scheduler/diagnostics?limit=50&due_soon_seconds=300
POST /v1/scheduler/leases/acquire
POST /v1/scheduler/leases/renew
POST /v1/scheduler/leases/release
GET  /v1/scheduler/leases
```

`/v1/scheduler/diagnostics` is read-only (`side_effect=none`) and summarizes
overdue, due-soon, disabled, trigger, tier, and channel counts. It is also
included in `/v1/runtime-overview` as the `Scheduler Jobs` card. The Python
runtime overview dashboard consumes that Go aggregate first and only calls this
diagnostics endpoint as a read-only fallback when the aggregate is unavailable.
Scheduler lease list responses never expose raw lease tokens; acquire/renew/
release only return a token to the current holder.
Scheduler job upsert/delete only mutate runtime scheduler state and do not
trigger tick execution, AI calls, outbox creation, or platform sends.
`/v1/scheduler/jobs/{job_id}/complete` requires the current execution lease
holder and token, then performs either `action=reschedule` with a full job body
or `action=delete`, and releases the lease only after the state write succeeds.
Python scheduler startup recovery also uses the single-job CRUD endpoints: missed
recurring jobs are advanced and persisted through upsert, while expired one-shot
jobs beyond the grace window are deleted through the delete endpoint. This
recovery reconciliation does not execute jobs, acquire scheduler execution
leases, call AI, create outbox records, or send platform messages.

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

Inspect group-level knowledge pipeline diagnostics:

```text
GET /v1/knowledge-pipeline-diagnostics?limit=50&stale_after_seconds=900
```

This read-only endpoint is the Go-owned knowledge control-plane view for each
observe-only QQ group target. It correlates observe capture readiness,
group-level `group_memory_extract` / `rag_ingest` backlog, knowledge
checkpoints, and Python worker coverage. It is intended to answer whether a
group pipeline is ready, warning, or blocked without inspecting multiple
endpoints manually. It also compares inbox `metadata.seq` against checkpoint
cursor and derives checkpoint `age_seconds`, so the runtime can expose
per-group lagging, stagnant, stalled, and stale/expired stage lease pipelines.
Configured per-group RAG dataset bindings synced from Python config also appear
here before any runtime `rag_ingest` job or checkpoint exists, as
`configured_dataset_not_started`. Each dataset also includes a Go-derived
`rag_index_state` based on checkpoint snapshot metadata, so empty indexes,
missing snapshots, source-lagging indexes, and ready indexes can be inspected
without calling RAGFlow. It does not upload to RAGFlow, execute Python workers,
or change memory/RAG strategy.

Inspect the Go-owned runtime overview aggregate:

```text
GET /v1/runtime-overview?limit=200&event_limit=50&stale_after_seconds=900
```

This read-only endpoint combines delivery adapter diagnostics, queue backend
state, runtime config diagnostics, runtime worker diagnostics, send ledger
metrics, inbox metrics, agent job metrics, outbox metrics, and knowledge worker
diagnostics into the same summary/card shape consumed by the Python dashboard.
`Agent Job Pressure` highlights job-type backlog pressure separately from
dead-letter and event throughput so knowledge/RAG buildup can be seen without
inspecting multiple endpoints manually. `Agent Job Worker Coverage` correlates
that pressure with Python worker heartbeat coverage, so backlog can be
distinguished between “worker exists and is healthy”, “worker stale/failed”, and
“no active worker available”. `Agent Job Capacity` summarizes the read-only
`/v1/agent-job-capacity/plan`, turning pressure plus worker coverage into
operational recommendations without starting workers, changing concurrency,
leasing jobs, acknowledging MQ messages, or executing AI work. `Knowledge
Pipelines` summarizes the same control
plane one level closer to the user workflow: per observe-only QQ group target,
combining capture, knowledge jobs, checkpoints, worker coverage, source-seq
lag, checkpoint-age diagnostics, stage lease freshness, configured dataset
bindings, per-dataset RAG state, and the last successful `rag_ingest`
snapshot plus derived RAG index readiness.
`Knowledge Planner` summarizes the read-only planner preview from
`/v1/knowledge-job-planner/preview`, including planned observe-only QQ targets,
groups, `group_memory_extract` jobs, and `rag_ingest` jobs before real planner
admission is enabled.
`Agent Job External Lease` summarizes the read-only
`/v1/agent-job-external-lease/readiness` gate, including result-ack readiness,
strict lease token, execution owner/scope and Python worker coverage blockers
before generic jobs use NATS external lease result acknowledgement.
`Agent Job External Lease Plan` summarizes the read-only
`/v1/agent-job-external-lease/plan`, including current, desired and recommended
execution owner, decision and blocker count before generic AgentJob queue
acknowledgement is moved to NATS result-ack.
`Delivery Smoke` summarizes the read-only `/v1/delivery-smoke/readiness`
matrix, using runtime-configured default smoke cases and channel aliases. It
checks dispatch planning and adapter support for QQ/Telegram send paths without
sending platform messages, leasing outbox deliveries, or invoking Python AI.
`Outbound Cutover` summarizes the read-only `/v1/outbound-cutover/plan`,
including current, desired and recommended execution owner, decision and blocker
count before QQ/NapCat delivery ownership is moved to Go local outbox worker or
NATS external lease.
`Media Asset Content` summarizes `/v1/media-assets/content-diagnostics`, showing
how many recent attachments are content-ready versus forbidden, unavailable,
disabled, or errored. It is only an access-control and local-content readiness
view; each item links to `/v1/media-assets/content-access-plan` for single-asset
drilldown. `/v1/media-assets/content-recovery-plan` explains disabled,
forbidden, unavailable, or probe-error recovery steps and future executor scope
without downloading, restoring, streaming, parsing, or invoking AI.
`/v1/media-assets/content-recovery/preflight` then binds an actionable recovery
candidate to `media_asset_content/recover_content` control mutation preflight
and an active operator approval; it still does not create approval/mutation
records or execute any download/restore/cache operation. The paired
`POST /v1/media-assets/content-recovery` endpoint performs the controlled
HTTP/HTTPS download/cache step after approval and audit checks pass. OCR, VLM,
file parsing, and semantic extraction remain Python AI worker responsibilities.
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
