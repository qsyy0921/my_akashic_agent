# Agent Gateway Specs

## Scope

The Go agent gateway owns platform messaging infrastructure:

- multi-account QQ connection;
- normalized message schema;
- provenance classification;
- signed bot-to-bot protocol;
- loop prevention;
- queue routing;
- audit events.

It does not own LLM reasoning, memory consolidation, RAG ranking, or tool
execution.

## Specs

- `001-message-envelope.md`: normalized inbound/outbound message schema.
- `002-provenance-and-bot-protocol.md`: human, self echo, peer bot, and signed
  bot-to-bot messages.
- `003-loop-guard.md`: TTL, budget, nonce, duplicate-content, and cooldown rules.
- `004-queue-and-routing.md`: NATS subjects, consumers, retries, and dead letters.
- `005-go-package-structure.md`: Go module package layout and dependency rules.
- `006-outbox-delivery-retry.md`: Go-owned outbound delivery state, retry, and
  dead-letter contract.
- `007-media-file-registry.md`: Go-owned QQ/TG media/file asset ids, metadata,
  retention, and safe dashboard access rules.
- `008-agent-job-orchestration.md`: Go-owned lifecycle, leasing, retry, and
  dead-letter control plane for image/RAG/memory jobs.
- `009-inbox-raw-message-store.md`: Go-owned raw observed/inbound message store
  for replay, dashboard queries, and group-memory source citations.
- `010-knowledge-checkpoints.md`: Go-owned knowledge ingestion cursors for
  RAGFlow and future memory/RAG workers.
- `011-knowledge-worker-diagnostics.md`: Python knowledge worker diagnostics
  are surfaced through Go-owned job/checkpoint status without moving AI
  extraction logic into Go.
- `012-agent-job-event-stream.md`: Go-owned durable lifecycle event stream for
  generic agent jobs.
- `013-proactive-scheduling-state.md`: Go-owned deterministic proactive
  scheduling state while Python keeps semantic candidate selection.
- `013-rag-eval-jobs.md`: Go-owned `rag_eval` lifecycle with Python eval
  worker execution.
- `014-python-proactive-state-runtime-bridge.md`: Python bridges proactive
  runtime state into Go-owned persistence while keeping AI behavior in Python.
- `014-runtime-dashboard-overview.md`: read-only dashboard aggregation for
  runtime health, worker leases, stale jobs, dead letters, checkpoint lag, and
  job event stream.
- `015-delivery-dispatch-plan.md`: Go-owned outbound dispatch planning before
  Python compatibility workers perform platform sends.
- `016-telegram-delivery-adapter.md`: Go-owned Telegram HTTP Bot API sender
  behind the delivery dispatch boundary.
- `017-runtime-backed-message-push.md`: Python `message_push` enqueue path into
  Go `/v1/outbound` and outbox workers for supported channels.
- `018-onebot-napcat-delivery-adapter.md`: Go-owned OneBot/NapCat HTTP or
  WebSocket sender for env-gated QQ private/group text, image, and file
  dispatch.
- `019-rag-eval-dashboard.md`: read-only dashboard panel for Go-owned
  `rag_eval` quality gates, metrics, trend points, and per-question results.
- `020-outbox-event-stream.md`: Go-owned durable lifecycle event stream for
  outbox delivery state transitions.
- `021-external-queue-backend.md`: external queue backend selection and
  migration phases from local state-store leasing to NATS JetStream first,
  with bounded concurrent consumers and Redis/RabbitMQ kept behind
  provider-neutral ports.
- `022-send-ledger-metrics.md`: Go-owned recent-send ledger metrics for
  bot-to-bot loop guard observability.
- `023-observe-target-diagnostics.md`: Go-owned runtime diagnostics for
  configured observe-only QQ group targets synced from Python config.
- `024-receiver-status-diagnostics.md`: Go-owned runtime diagnostics for
  Python platform receiver lifecycle states, including QQ connected status and
  Telegram polling conflict suspension.
- `025-receiver-lease-control.md`: Go-owned receiver lease control for
  single-instance long-running platform polling loops.
- `026-observe-capture-diagnostics.md`: Go-owned read-only diagnostics for
  observe-only QQ group text/image/file capture and safe media content coverage.
- `027-receiver-status-heartbeat.md`: durable receiver status storage and
  Python QQ/Telegram heartbeat reporting for restart-safe connectivity
  diagnostics.
- `028-receiver-lease-persistence.md`: durable receiver lease storage and
  Telegram reacquire behavior after `agent-runtime` restarts.
- `029-observe-capture-activity-inference.md`: Go observe-capture diagnostics
  infer effective receiver connectivity from recent observe-only inbox events.
- `030-inbound-dedupe-runtime.md`: Go-owned TTL/persistent duplicate detection
  for inbound platform message ids, starting with Telegram receive paths.
- `031-agent-worker-status.md`: Go-owned liveness/status registry for Python
  AI workers while Python keeps model and tool execution.
- `032-proactive-anyaction-quota.md`: Go-owned deterministic AnyAction quota
  window state for proactive admission, with Python retaining probability logic.
- `033-proactive-seen-and-rejection-state.md`: Go-owned proactive source item
  seen dedupe and rejection cooldown state, with Python retaining semantic
  candidate extraction and tick logs.
- `034-proactive-retention-cleanup.md`: Go-owned TTL cleanup for proactive
  deterministic runtime state, while Python keeps SQLite fallback cleanup.
- `035-proactive-bg-context-global-mark.md`: Go-owned global mark for proactive
  background-context main-topic send timestamp.
- `036-scheduler-job-store.md`: Go-owned durable snapshot store for Python
  scheduler jobs, while Python keeps tick loop and AI execution.
- `037-scheduler-job-diagnostics.md`: Go-owned read-only scheduler snapshot
  diagnostics and runtime overview card.
- `038-scheduler-execution-lease.md`: Go-owned per-job scheduler execution
  lease/fencing, while Python keeps tick loop and job execution.
- `039-scheduler-job-crud.md`: Go-owned single scheduler job upsert/delete for
  tool-driven add/cancel, reducing snapshot overwrite risk.
- `040-scheduler-completion-mutation.md`: Go-owned lease-fenced scheduler
  completion mutation for recurring reschedule and one-shot delete.
- `041-scheduler-recovery-reconciliation.md`: Python startup scheduler recovery
  persists recurring misfire advancement and expired one-shot deletion through
  existing Go-owned scheduler CRUD.
- `042-proactive-drift-state.md`: Go-owned proactive drift skill state and
  recent-run summary, with Python retaining skill file scanning and AI/tool
  execution.
- `043-proactive-tick-log-state.md`: Go-owned proactive tick start/finish/step
  audit state, with Python retaining proactive behavior and SQLite dashboard
  mirror.
- `044-dashboard-proactive-tick-log-runtime-fallback.md`: Dashboard proactive
  tick log read fallback to Go runtime when SQLite mirror has no matching
  records, plus Go query pagination/sort filters.
- `045-agent-worker-status-lease-fencing.md`: Go-owned lease/fencing for
  Python AI worker status ownership, preventing active same-worker-id status
  overwrites from another process instance.
- `046-agent-worker-status-heartbeat-renewal.md`: Python AI workers renew
  Go-owned worker-status leases during long-running jobs without changing
  AgentJob lease semantics.
- `047-external-lease-execution-diagnostics.md`: Go-owned diagnostics for
  external queue lease execution disposition, reason, work-kind counters, and
  bounded recent samples.
- `048-runtime-overview-external-lease-diagnostics.md`: Runtime overview summary
  and card aggregation for external lease execution diagnostics.
- `049-outbox-account-pressure-diagnostics.md`: Go-owned outbox account pressure
  diagnostics for queued/dispatching delivery backlog before actual rate-limit
  enforcement.
- `050-outbox-account-rate-limit.md`: Go local outbox delivery worker
  account-level send throttling with lease-time blocked-account skip.
- `051-external-lease-outbox-account-rate-limit.md`: NATS external lease outbox
  executor uses the same Go account throttling policy before leasing/sending.
- `052-queue-provider-capability-diagnostics.md`: Queue backend provider
  capability matrix for NATS-first MQ selection, concurrent consumers, and
  future Redis/RabbitMQ adapter boundaries.
- `053-runtime-overview-queue-provider-capability.md`: Runtime overview summary
  fields for selected queue provider capability and NATS-first MQ visibility.
- `054-external-lease-local-worker-conflict-gate.md`: External lease cutover
  gate blocks when the local outbox delivery worker is still enabled.
- `055-queue-execution-owner-diagnostics.md`: Queue backend and runtime
  overview expose current outbox and agent_job execution owners.
- `056-agent-job-pressure-diagnostics.md`: Go-owned AgentJob backlog and
  pressure diagnostics for pending/active job-type buildup, especially
  knowledge and RAG workers.
- `057-agent-job-worker-coverage-diagnostics.md`: Runtime overview correlates
  Go-owned AgentJob pressure with Python worker heartbeat coverage so backlog
  can be explained by missing, stale, or failed worker capacity.
- `058-knowledge-pipeline-diagnostics.md`: Go-owned group-level knowledge
  pipeline diagnostics correlate observe capture, knowledge jobs, checkpoints,
  and worker coverage for each observe-only QQ target.
- `059-knowledge-pipeline-source-lag-diagnostics.md`: Go-owned group-level
  knowledge pipeline diagnostics compare inbox source seq against checkpoint
  cursor to expose lagging and stalled pipelines.
- `060-knowledge-pipeline-checkpoint-age-diagnostics.md`: Go-owned knowledge
  pipeline diagnostics add checkpoint age and stagnant-state visibility so
  long-unmoved memory/RAG checkpoints can be separated from short-lived lag.
- `061-knowledge-pipeline-job-lease-freshness-diagnostics.md`: Go-owned
  knowledge pipeline diagnostics add per-group job lease freshness so stale or
  expired `group_memory_extract` / `rag_ingest` execution can be seen directly.
- `062-knowledge-pipeline-rag-dataset-state-diagnostics.md`: Go-owned
  knowledge pipeline diagnostics add per-dataset RAG state so dataset-specific
  `rag_ingest` lag and lease degradation can be inspected directly.
- `063-knowledge-pipeline-configured-rag-dataset-bindings.md`: Python observe
  target sync exposes configured per-group RAG dataset bindings so Go
  diagnostics can show datasets that should exist before jobs/checkpoints have
  started.
- `064-knowledge-pipeline-rag-ingest-snapshot-diagnostics.md`: Python writes a
  bounded successful `rag_ingest` snapshot into checkpoint metadata and Go
  surfaces it as structured per-dataset diagnostics.
- `065-go-owned-knowledge-job-planner.md`: Go runtime owns recurring
  observe-only knowledge job admission while Python knowledge workers retain
  lease-based execution.
- `066-knowledge-job-planner-preview-diagnostics.md`: Go exposes a read-only
  planner preview so observe-only knowledge job admission can be preflighted
  without creating AgentJob records.
- `067-runtime-overview-knowledge-planner-preview.md`: Runtime overview
  aggregates the read-only planner preview so dashboard/operator entrypoints can
  see planned knowledge jobs before enabling real admission.
- `068-knowledge-job-planner-readiness.md`: Go exposes a read-only planner
  readiness gate that combines preview, runtime worker state and Python
  knowledge worker status before enabling real admission.
- `069-runtime-overview-knowledge-planner-readiness.md`: Runtime overview
  aggregates knowledge planner readiness so dashboard/operator entrypoints can
  see admission blockers without mutating jobs or config.
- `070-knowledge-pipeline-rag-index-state-diagnostics.md`: Go derives
  per-dataset RAG index readiness from checkpoint metadata without calling
  RAGFlow or changing Python RAG execution.
- `071-outbound-cutover-readiness.md`: Go exposes a read-only outbound cutover
  preflight that combines OneBot config, delivery smoke readiness, queue
  execution owner and runtime worker state before enabling Go platform sends.
- `072-outbound-cutover-plan.md`: Go exposes a read-only outbound cutover plan
  with required checks, enable steps, verification endpoints, rollback steps,
  blockers and explicit Go/Python delivery boundary before operator cutover.
- `073-runtime-overview-outbound-cutover-plan.md`: Runtime overview aggregates
  the outbound cutover plan into summary/card/detail fields for dashboard and
  operator entrypoints without mutating delivery state.
- `074-agent-job-external-lease-readiness.md`: Go exposes a read-only
  `agent_job` external lease result-ack readiness gate combining queue gate,
  strict lease tokens, runtime config and Python worker coverage.
- `075-runtime-overview-agent-job-external-lease-readiness.md`: Runtime
  overview aggregates the `agent_job` external lease readiness gate into
  summary/card/detail fields without mutating queue, worker or AI execution
  state.
- `076-agent-job-external-lease-plan.md`: Go exposes a read-only `agent_job`
  external lease result-ack cutover plan with required checks, env hints,
  verification steps and rollback steps while Python keeps AI execution.
- `077-runtime-overview-agent-job-external-lease-plan.md`: Runtime overview
  aggregates the `agent_job` external lease plan into summary/card/detail
  fields without mutating queue, worker or AI execution state.
- `078-agent-job-capacity-plan.md`: Go exposes a read-only AgentJob capacity
  plan that turns pressure plus Python worker coverage into operational
  recommendations without scheduling or AI execution side effects.
- `079-runtime-overview-agent-job-capacity-plan.md`: Runtime overview
  aggregates the AgentJob capacity plan into summary/card/detail fields without
  starting workers, scheduling jobs or mutating queue/config state.
- `080-media-asset-content-diagnostics.md`: Go exposes read-only media asset
  content readiness diagnostics for frontend attachment troubleshooting without
  changing content access policy or running multimodal AI.
- `081-runtime-overview-media-asset-content.md`: Runtime overview aggregates
  media asset content readiness so dashboard and operators can see attachment
  display health without running OCR/VLM or changing file access policy.
- `089-dashboard-runtime-control-plane-details.md` through later runtime
  overview/dashboard specs keep Python as a read-only presentation layer for
  Go-owned control-plane details.
- `103-media-asset-retention-diagnostics.md`: Go exposes read-only media asset
  retention diagnostics with TTL and cleanup-due advisory fields.
- `104-runtime-overview-media-retention.md`: Runtime overview aggregates media
  retention diagnostics without deleting metadata/files or invoking AI.
- `105-media-asset-retention-plan.md`: Go exposes a read-only retention cleanup
  plan with approval/audit/verification/rollback steps.
- `106-runtime-overview-media-retention-plan.md`: Runtime overview exposes the
  retention cleanup plan as a read-only card/detail.
- `107-media-retention-control-policy.md`: Go control mutation policy allowlists
  media retention cleanup as an explicit target/action.
- `108-media-retention-cleanup-preflight.md`: Go validates cleanup candidates
  and active operator approval before metadata cleanup, with `side_effect=none`.
- `109-media-retention-metadata-cleanup.md`: Go executes approved metadata-only
  media cleanup and records control mutation audit; local files and AI pipelines
  remain untouched.
- `110-runtime-overview-media-retention-cleanup.md`: Runtime overview aggregates
  media retention cleanup readiness and recent cleanup audits.
- `112-media-asset-content-access-plan.md`: Go exposes a read-only single-asset
  content access plan with ready/reason/blockers and no content streaming.
- `113-dashboard-media-content-access-plan.md` through
  `116-media-content-access-plan-contract.md`: Python dashboard proxies and
  contracts the Go-owned content access plan without taking policy ownership.
- `117-media-content-diagnostics-access-plan-endpoint.md`: Go diagnostics items
  include the single-asset access-plan endpoint, and Python dashboard preserves
  it for drilldown.
- `082-runtime-overview-delivery-smoke.md`: Runtime overview aggregates
  delivery smoke readiness so dashboard and operators can inspect QQ/Telegram
  send-path cutover gates without sending messages or invoking AI.
- `083-dashboard-runtime-overview-new-fields.md`: Python dashboard normalizes
  the latest Go-owned runtime overview delivery-smoke and media-asset-content
  fields without taking ownership of runtime state or AI processing.
- `084-knowledge-job-planner-cutover-plan.md`: Go exposes a read-only
  knowledge job planner cutover plan and runtime overview aggregate so
  observe-only memory/RAG admission can be enabled or rolled back deliberately
  without creating jobs or executing AI.
- `085-dashboard-knowledge-planner-cutover-plan.md`: Python dashboard
  normalizes the Go-owned knowledge planner cutover plan from runtime overview
  without taking ownership of admission, jobs, worker startup or AI execution.
- `086-dashboard-runtime-control-plane-details.md`: Python dashboard normalizes
  Go-owned AgentJob capacity, external lease and outbound cutover details from
  runtime overview while staying a read-only projection.
- `087-queue-topology-read-model.md`: Go exposes a read-only queue topology
  view that maps provider capability, execution owners and external lease gates
  without executing MQ or AI work.
- `088-runtime-overview-queue-topology.md`: Runtime overview and Python
  dashboard aggregate the Go-owned queue topology read model as a stable
  read-only status surface.
- `089-agent-job-priority-plan.md`: Go exposes a read-only AgentJob priority
  plan from pressure and Python worker coverage without scheduling or AI side
  effects.
- `090-runtime-overview-agent-job-priority-plan.md`: Runtime overview aggregates
  the read-only AgentJob priority plan.
- `091-operator-approval-ledger.md`: Go-owned operator approval audit ledger
  for control-plane plans.
- `092-operator-approval-check.md`: Go-owned active approval check before
  control-plane mutations.
- `093-control-mutation-audit-ledger.md`: Go-owned audit ledger for planned,
  applied, failed, or rolled-back control mutations.
- `094-runtime-overview-control-audit.md`: Runtime overview aggregates operator
  approvals and control mutation audit status.
- `095-dashboard-control-audit-detail.md`: Python dashboard normalizes
  Go-owned control audit detail as a read-only projection.
- `096-control-mutation-preflight.md`: Go validates approval-bound mutation
  preflight without executing control changes.
- `097-control-mutation-policy.md`: Go-owned control mutation policy prevents
  unsupported target/action strings from becoming executable.
- `098-control-mutation-policy-api.md`: Go exposes the control mutation
  allowlist as a read-only API.
- `099-runtime-overview-control-mutation-policy.md`: Runtime overview
  aggregates the Go-owned mutation policy.
- `100-dashboard-control-mutation-policy.md`: Python dashboard normalizes
  Go-owned mutation policy detail without owning the allowlist.
- `101-receiver-lease-cleanup.md`: Go-owned receiver lease cleanup diagnostics
  and state hygiene.
- `102-runtime-overview-receiver-lease-cleanup.md`: Runtime overview aggregates
  receiver lease cleanup status.
- `111-dashboard-media-retention-cleanup.md`: Python dashboard normalizes
  Go-owned media retention cleanup detail as read-only state.
- `114-dashboard-media-content-access-plan-proxy.md`: Python dashboard proxies
  the Go-owned media content access plan without copying policy.
- `115-dashboard-message-media-access-plan-link.md`: Dashboard message media
  entries expose direct access-plan links for operator drilldown.
- `118-agent-runtime-media-api-docs.md`: Agent runtime README and SDD index
  document current media asset APIs and side-effect boundaries.
- `119-go-contract-test-media-access-plan.md`: Go contract tests validate the
  shared media content access plan fixture.
- `120-sdd-spec-index-guard.md`: Repository tests require agent-gateway specs
  to be referenced by exact filename in this index.
- `121-go-runtime-package-guard.md`: Go architecture tests require runtime
  source files to stay under intentional DDD/hexagonal top-level roots.
- `122-media-content-access-plan-url-fields.md`: Go access plans expose
  runtime/dashboard URL hints while Python remains a read-only proxy.
- `123-open-issues-register.md`: SDD adds a canonical unresolved issue ledger
  separate from current-iteration TODO and future backlog.
- Agent architecture boundary specs live under `../agent-architecture/`.
