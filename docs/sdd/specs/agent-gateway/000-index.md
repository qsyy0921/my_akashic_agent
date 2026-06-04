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
- `179-dashboard-outbox-pressure-table.md`: runtime overview dashboard
  structured read model for Go-owned outbox account pressure diagnostics.
- `180-dashboard-outbox-metrics-table.md`: runtime overview dashboard
  structured read model for Go-owned outbox delivery throughput and dead-letter diagnostics.
- `181-dashboard-send-ledger-metrics-table.md`: runtime overview dashboard
  structured read model for Go-owned send ledger repeat and recent-send diagnostics.
- `182-dashboard-inbox-metrics-table.md`: runtime overview dashboard
  structured read model for Go-owned inbox capture, observe-only, and recent-event diagnostics.
- `183-dashboard-inbound-dedupe-metrics-table.md`: runtime overview dashboard
  structured read model for Go-owned inbound dedupe scope and duplicate-seen diagnostics.
- `184-dashboard-observe-targets-table.md`: runtime overview dashboard
  structured read model for Go-owned observe-target policy and observe-only coverage diagnostics.
- `185-dashboard-observe-capture-table.md`: runtime overview dashboard
  structured read model for Go-owned observe-capture coverage and media-readiness diagnostics.
- `186-dashboard-delivery-adapters-table.md`: runtime overview dashboard
  structured read model for Go-owned delivery-adapter configuration visibility.
- `187-dashboard-agent-job-metrics-table.md`: runtime overview dashboard
  structured read model for Go-owned AgentJob throughput, pressure, and dead-letter diagnostics.
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
- `142-local-knowledge-planner-bringup.md`: repo-local runtime start entrypoint
  can explicitly enable the Go knowledge planner, and Python records skip-legacy
  evidence when Go owns recurring admission.
- `143-native-napcat-rich-media-comparison.md`: repo-local native NapCat/OneBot
  rich-media smoke replays the same streamed upload path as Akashic so QQ
  `rich media transfer failed` can be classified as platform/session parity or
  adapter drift before default Go cutover.
- `144-text-only-outbox-cutover-gate.md`: Python compatibility outbox worker
  backs off while Go local outbox ownership is active, making repo-local
  text-only QQ cutover real instead of partial.
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
- `137-runtime-local-bringup-post-migration.md`: local migrated workspace bring-up
  and runtime endpoint verification after session transfer.
- `138-qq-live-send-smoke-runbook.md`: repo-owned private-text QQ live smoke
  path through Go `delivery-dispatch/send`.
- `139-qq-group-media-live-smoke-runbook.md`: repo-owned QQ group text/image/file
  live smoke path plus WebSocket media-frame tolerance requirement.
- `140-local-outbox-worker-success-state.md`: local outbox worker uses the lease
  transition as the only dispatching state mutation so automatic success is not
  mis-recorded as dead-lettered.
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
- `124-open-issues-registry-guard.md`: SDD governance tests enforce
  structured open issue rows with stable ids and status values.
- `125-media-content-recovery-plan.md`: Go exposes a read-only media content
  recovery plan and Python dashboard proxies it without downloading,
  restoring, parsing or invoking AI.
- `126-media-recovery-plan-drilldown-url.md`: Go diagnostics and Python
  dashboard message media entries expose deterministic recovery-plan drilldown
  links without fetching the plan during list/detail rendering.
- `127-runtime-overview-media-content-table.md`: Runtime overview dashboard
  renders Go-owned media content diagnostics as a read-only table with
  content/access/recovery links while preserving JSON fallback.
- `128-runtime-overview-queue-topology-table.md`: Runtime overview dashboard
  renders Go-owned queue topology work kinds and execution/ack owners as a
  read-only table while preserving JSON fallback.
- `129-runtime-overview-control-audit-table.md`: Runtime overview dashboard
  renders Go-owned operator approval and control mutation audit detail as
  read-only tables while preserving JSON fallback.
- `130-runtime-overview-control-mutation-policy-table.md`: Runtime overview
  dashboard renders Go-owned control mutation target/action allowlist as a
  read-only table while preserving JSON fallback.
- `131-media-content-recovery-preflight.md`: Go exposes a read-only
  approval-bound media content recovery preflight for future recovery/download
  executors without restoring files or invoking AI.
- `132-dashboard-media-recovery-preflight-proxy.md`: Python dashboard proxies
  the Go-owned media content recovery preflight and exposes deterministic
  media asset drilldown URLs without taking policy or executor ownership.
- `133-media-content-diagnostics-recovery-preflight-endpoint.md`: Go media
  content diagnostics and runtime overview dashboard expose recovery preflight
  drilldown endpoints without invoking recovery execution.
- `134-media-content-recovery-executor.md`: Go exposes an approval-bound media
  content recovery executor that downloads HTTP/HTTPS media into a local cache,
  updates the registry, and records control mutation audit while leaving OCR/VLM
  and semantic parsing to Python.
- `135-runtime-overview-media-content-recovery.md`: Runtime overview exposes a
  focused read-only `media_asset_content_recovery` card/detail from Go control
  mutation audits without executing recovery or invoking Python AI.
- `136-dashboard-media-content-recovery-table.md`: Runtime overview dashboard
  renders `media_asset_content_recovery` as read-only KPI/link/audit tables
  while preserving raw JSON fallback and avoiding recovery execution.
- `137-runtime-local-bringup-post-migration.md`: The migrated workspace keeps a
  repo-owned local launcher and read-only preflight contract for bringing Go
  `agent-runtime` back on `127.0.0.1:8780` without faking Telegram readiness or
  triggering live send cutover.
- `138-qq-live-send-smoke-runbook.md`: Repo-owned QQ private-text live smoke
  entrypoint uses Go outbox plus `delivery-dispatch/send` for explicit local
  verification without enabling full outbox cutover.
- `139-qq-group-media-live-smoke-runbook.md`: repo-owned QQ group text/image/file
  smoke records real text success and rich-media blockers without faking cutover.
- `140-local-outbox-worker-success-state.md`: local outbox worker uses the lease
  path correctly and can become the real execution owner for private-text smoke.
- `141-onebot-websocket-stream-media-staging.md`: OneBot WebSocket local media
  uses `upload_file_stream` staging so Docker path issues are separated from
  real NapCat / QQ rich-media platform failures.
- `145-native-napcat-cross-group-rich-media-verification.md`: Native NapCat
  rich-media parity is repeated across additional QQ groups to distinguish a
  single-group anomaly from a current session-wide blocker.
- `146-napcat-session-refresh-rich-media-recheck.md`: Restarting the NapCat
  container is used as the minimal session refresh attempt before escalating to
  manual QQ re-login or session replacement.
- `147-dual-account-rich-media-split-verification.md`: Native and Akashic
  rich-media behavior is compared across both QQ accounts to split account-
  specific file failures from cross-account image failures.
- `148-account-kind-outbox-cutover-gate.md`: Go local outbox execution can be
  narrowed by both delivery kind and QQ account so second-account file sending
  is enabled without globally opening first-account rich-media.
- `149-native-private-rich-media-route-verification.md`: Native and Akashic
  private rich-media verification distinguishes cross-route image failure from
  first-account group-file-only failure.
- `150-account-conversation-kind-outbox-cutover-gate.md`: Go local outbox
  execution can be narrowed by account, conversation type, and delivery kind so
  first-account private file is enabled without opening first-account group
  file or any image route.
- `151-knowledge-planner-cutover-verification-runbook.md`: A repo-owned
  read-only verification script re-checks live runtime and recent logs to prove
  Go still owns recurring knowledge admission while Python only executes jobs.
- `152-telegram-backend-verification-runbook.md`: A repo-owned read-only
  verification script distinguishes Telegram token absence from later `getMe`
  or receiver-chain failures.
- `153-go-migration-goal-verification-runbook.md`: A repo-owned goal audit
  script aggregates current QQ outbox scope, optional native rich-media probe,
  Telegram backend state, knowledge planner health, and remaining residual
  classification into one live JSON result for end-of-turn goal updates.
- `154-go-outbox-scope-live-verification-runbook.md`: A repo-owned live smoke
  script proves which outbox routes currently auto-succeed under Go owner and
  which routes remain safely gated, and can be embedded into the goal verifier
  only when explicitly requested.
- `155-goal-verification-hardening.md`: Goal verification tolerates structured
  OneBot websocket status fields, local runtime bring-up can explicitly enable
  the outbox worker, and Telegram verification now proves config-declared but
  unresolved token injection.
- `156-local-runtime-account-channel-mapping.md`: The repo-local runtime
  launcher injects per-account QQ channel mapping so second-account outbox smoke
  uses the correct OneBot alias during gated Go cutover verification.
- `157-observe-only-group-reply-hard-block.md`: Delivery dispatch refuses exact
  observe-only QQ group routes with `reply_allowed=false`, making silent group
  observation a runtime invariant instead of a config-only convention.
- `158-goal-verifier-observe-only-silence.md`: The unified repo-owned goal
  verifier includes live observe-only QQ group silence checks and reports stable
  blockers if the hard block or private-route readiness regresses.
- `159-qq-group-send-manual-toggle.md`: QQ group sending is now guarded by a
  manual global toggle across both Go runtime and Python channel paths, with
  the current local default set to disabled while private routes stay available.
- `160-goal-verifier-scheduler-proactive-live-checks.md`: Unified goal
  verification now uses live scheduler/proactive/dashboard evidence instead of
  `not_current_turn` placeholders for those residual areas.
- `161-agent-job-external-lease-live-verifier.md`: A repo-owned live verifier
  explains whether agent-job external lease result-ack is blocked by config,
  ownership, smoke evidence, or worker coverage.
- `162-media-recovery-boundary-live-verifier.md`: A repo-owned live verifier
  classifies the current media recovery boundary from runtime diagnostics,
  recovery plans, preflight, and dashboard evidence, including whether the
  active gap is operator content-root configuration or a broader private-source
  executor/read-model follow-up.
- `163-dashboard-media-recovery-runtime-overview-read-model.md`: The dashboard
  runtime-overview reader now preserves `media_asset_content_recovery`
  summary/detail and synthesizes the missing card, while using a realistic
  timeout so live Go overview data does not silently degrade to fallback.
- `164-agent-worker-status-restart-takeover.md`: Python worker-status reporting
  can perform a single controlled stale-instance takeover after `main.py`
  restart, while Go keeps default conflict fencing for active same-worker-id
  leases.
- `165-mq-adapter-boundary-dashboard-read-model.md`: The dashboard
  runtime-overview reader now preserves queue-backend provider capability
  details, and repo-owned live verification proves NATS remains the only
  implemented recommended external MQ while Redis Streams and RabbitMQ stay
  planned-only boundaries.
- `166-dashboard-agent-job-external-lease-table.md`: The runtime overview
  dashboard now renders `agent_job_external_lease_readiness` and
  `agent_job_external_lease_plan` as structured read-only drilldowns instead of
  leaving them raw-JSON-only.
- `167-dashboard-runtime-plan-drilldowns.md`: The runtime overview dashboard
  now renders capacity/priority/knowledge-cutover/outbound-cutover plan details
  as structured read-only drilldowns instead of leaving them raw-JSON-only.
- `168-dashboard-delivery-smoke-readiness-table.md`: The runtime overview
  dashboard now renders `delivery_smoke_readiness` as a structured read-only
  drilldown instead of leaving smoke case inspection to raw JSON.
- `169-dashboard-queue-backend-table.md`: The runtime overview dashboard now
  renders `queue_backend` as a structured read-only drilldown with provider,
  execution-owner, and capability-matrix visibility instead of leaving MQ
  boundary inspection to raw JSON.
- `170-dashboard-receiver-statuses-table.md`: The runtime overview dashboard
  now renders `receiver_statuses` as a structured read-only drilldown so QQ
  connectivity and Telegram receiver absence are visible without reading raw
  JSON.
- `171-dashboard-receiver-leases-table.md`: The runtime overview dashboard now
  renders `receiver_leases` as a structured read-only drilldown so single-
  instance receiver ownership and expired-lease state are visible without
  reading raw JSON.
- `172-dashboard-knowledge-pipelines-table.md`: The runtime overview dashboard
  now renders `knowledge_pipelines` as a structured read-only drilldown so
  observe capture warnings, knowledge freshness, and worker coverage are
  visible without reading raw JSON.
- `173-dashboard-external-lease-diagnostics-table.md`: The runtime overview
  dashboard now renders `external_lease_diagnostics` as a structured read-only
  drilldown so provider capability, queue mode, and result-ack support are
  visible without reading raw JSON.
- `174-dashboard-scheduler-jobs-table.md`: The runtime overview dashboard now
  renders `scheduler_jobs` as a structured read-only drilldown so trigger/tier
  distribution and recent scheduler samples are visible without reading raw
  JSON.
- `175-dashboard-agent-workers-table.md`: The runtime overview dashboard now
  renders `agent_workers` as a structured read-only drilldown so Python worker
  liveness, stale state, and lease activity are visible without reading raw
  JSON.
- `176-dashboard-runtime-workers-table.md`: The runtime overview dashboard now
  renders `runtime_workers` as a structured read-only drilldown so Go worker
  enablement, running state, and queue-boundary settings are visible without
  reading raw JSON.
- `177-dashboard-agent-job-worker-coverage-table.md`: The runtime overview
  dashboard now renders `agent_job_worker_coverage` as a structured read-only
  drilldown so expected-worker coverage and stale/failed worker state are
  visible without reading raw JSON.
- `178-dashboard-agent-job-pressure-table.md`: The runtime overview dashboard
  now renders `agent_job_pressure` as a structured read-only drilldown so
  pressure, throughput, and dead-letter baselines are visible without reading
  raw JSON.
- `188-dashboard-worker-leases-table.md`: The runtime overview dashboard now
  renders `worker_leases` as a structured read-only drilldown so lease
  diagnostics, checkpoint prefixes, and stale lease counts are visible without
  reading raw JSON.
- `189-dashboard-runtime-config-table.md`: The runtime overview dashboard now
  renders `runtime_config` as a structured read-only drilldown so runtime
  address, delivery toggles, adapter endpoints, worker flags, and sanitized
  environment visibility are available without reading raw JSON.
- `190-dashboard-knowledge-planner-preview-readiness-table.md`: The runtime
  overview dashboard now renders `knowledge_job_planner_preview` and
  `knowledge_job_planner_readiness` as structured read-only drilldowns so
  observe-only admission preview and planner/worker readiness are visible
  without reading raw JSON.
- `191-dashboard-media-asset-retention-plan-cleanup-table.md`: The runtime
  overview dashboard now renders `media_asset_retention_plan` and
  `media_asset_retention_cleanup` as structured read-only drilldowns so
  retention candidates, required steps, cleanup totals, and notes are visible
  without reading raw JSON.
- `192-dashboard-media-asset-retention-table.md`: The runtime overview
  dashboard now renders `media_asset_retention` as a structured read-only
  drilldown so retention totals, recent assets, and diagnostics notes are
  visible without reading raw JSON.
- `193-dashboard-dead-letters-table.md`: The runtime overview dashboard now
  renders `dead_letters` as a structured read-only drilldown so AgentJob and
  outbox dead-letter aggregates are visible without reading raw JSON.
- `194-dashboard-checkpoint-lag-table.md`: The runtime overview dashboard now
  renders `checkpoint_lag` as a structured read-only drilldown so lagged
  checkpoints and freshness diagnostics are visible without reading raw JSON.
- `195-dashboard-runtime-health-stale-jobs-table.md`: The runtime overview
  dashboard now renders `runtime_health` and `stale_jobs` as structured
  read-only drilldowns so health snapshot fields, health errors, worker stale
  diagnostics, and sampled stale jobs are visible without reading raw JSON.
- `196-dashboard-job-events-outbox-events-table.md`: The runtime overview
  dashboard now renders `job_events` and `outbox_events` as structured
  read-only drilldowns so event totals and recent job/outbox event rows are
  visible without reading raw JSON.
- `197-dashboard-rag-eval-failures-table.md`: The runtime overview dashboard
  now renders `rag_eval_failures` as a structured read-only drilldown so
  failure totals, sampled failure rows, and notes are visible without reading
  raw JSON.
- `198-scheduler-runtime-mutation-live-smoke.md`: An isolated temp-runtime
  verifier now proves Go-owned scheduler CRUD persistence, lease-fenced
  completion mutation, and Python recovery reconciliation without mutating the
  long-running local runtime.
- `199-agent-job-external-lease-temp-nats-smoke-verifier.md`: A repo-owned
  temp-NATS verifier now proves duplicate terminal ack and
  pending/running/succeeded result-ack flow for `agent_job` external lease
  without modifying the long-running local runtime.
- `200-queue-topology-boundary-live-verifier.md`: A repo-owned live verifier
  now proves `queue_topology`, `queue_backend`, and runtime-overview queue
  summary agree on provider recommendation, execution owners, ack owner, and
  external-lease readiness without executing MQ or AI work.
- `201-worker-control-executor-boundary-live-verifier.md`: A repo-owned live
  verifier now proves capacity/priority plan state, control-mutation policy,
  runtime workers, and runtime-overview summary agree that Go has the control
  plane but no worker-control executor yet.
- `202-qq-cutover-route-matrix-live-verifier.md`: A repo-owned live verifier
  now classifies the current QQ cutover routes into Go-owned scope, currently
  sendable scope, group-send policy blocks, and unresolved rich-media blocker
  routes.
- `203-dashboard-qq-cutover-route-matrix-table.md`: The runtime overview
  dashboard now renders `qq_cutover_route_matrix` as a structured read-only
  drilldown so QQ cutover scope, currently sendable routes, policy-blocked
  routes, and platform-blocker routes are visible without reading raw JSON.
- `204-agent-worker-status-fencing-live-smoke.md`: A repo-owned temp-runtime
  verifier now proves default worker-status lease fencing returns HTTP 409 for
  a second live instance and only allows takeover when
  `replace_existing_instance_id` matches the active lease owner.
- `205-agent-worker-status-heartbeat-live-smoke.md`: A repo-owned temp-runtime
  verifier now proves repeated heartbeats from the same worker instance keep
  `updated_at/lease_until` advancing and do not mark the worker stale.
- `206-agent-job-external-lease-cutover-preflight-live-verifier.md`: A
  repo-owned isolated verifier now proves both the current blocked
  state-store-owner semantics without external-lease flags and the ready
  NATS/result-ack owner semantics once temp runtime flags and active knowledge
  worker coverage are satisfied.
- `207-control-audit-boundary-live-verifier.md`: A repo-owned isolated verifier
  now proves operator-approval ledger, control-mutation audit ledger,
  approval-bound preflight, policy allowlist, and runtime-overview
  control-audit summaries stay aligned without introducing executor side
  effects.
- `208-dashboard-control-audit-boundary-live-verifier.md`: A repo-owned live
  verifier now proves dashboard runtime-overview preserves `control_audit` and
  `control_mutation_policy` summary/card/detail parity with the Go runtime
  overview on the current turn.
- `209-agent-worker-status-stale-cleanup-live-smoke.md`: Go now exposes an
  explicit stale worker-status cleanup mutation plus a repo-owned verifier that
  proves stale heartbeat residue can be removed without deleting active worker
  state.
- `210-goal-verifier-current-state-artifact.md`: The unified migration
  verifier now writes a canonical repo-local JSON artifact and exposes a stable
  `current_state` summary for current-turn cutover, Telegram, worker-stale, and
  dashboard-fallback evidence.
- `211-goal-verifier-migration-residuals.md`: The unified migration verifier
  now emits a machine-readable `migration_residuals` section for QQ, Telegram,
  external lease result-ack, scheduler, worker-control executors, media
  recovery, dashboard read-models, and MQ adapter boundary.
- `212-goal-verifier-migration-buckets.md`: The unified migration verifier now
  normalizes each residual entry into one of the four exact migration buckets
  and exposes a top-level bucket summary.
- `213-proactive-runtime-flow-live-smoke.md`: A repo-owned isolated verifier
  now proves Go-owned proactive deliveries, anyaction quota, seen/rejection
  cleanup, context-only, drift, bg-context, and tick-log state transitions in
  a temp runtime without triggering Python AI or platform sends.
- `214-dashboard-proactive-tick-logs-runtime-fallback-live-verifier.md`: A
  repo-owned isolated verifier now proves dashboard proactive tick-log
  list/detail/steps can fall back to Go runtime state when the SQLite mirror is
  empty, without writing fallback rows back into SQLite.
- `215-goal-verifier-dashboard-read-model-checks-alias.md`: The unified goal
  verifier now aliases `dashboard_read_models` under `checks.*` as well, so
  machine readers can consume the canonical dashboard evidence from either path
  without treating verified fields as missing.
- `216-media-asset-content-recovery-http-executor-live-smoke.md`: A repo-owned
  isolated verifier now proves approval-bound HTTP/HTTPS media recovery,
  dry-run no-write semantics, registry update, local content readback, and
  control-mutation audit in a temp Go runtime.
- `217-dashboard-knowledge-rag-state-boundary-live-verifier.md`: A repo-owned
  live verifier now proves dashboard runtime-overview preserves the Go
  `knowledge_pipelines` card/top-level detail and the current
  checkpoint-derived RAG dataset/index boundary.
- `218-goal-verifier-dashboard-knowledge-rag-retry-hardening.md`: The unified
  goal verifier now retries the dashboard knowledge/RAG live boundary verifier
  once before downgrading current-turn dashboard fallback evidence.
- `219-dashboard-media-asset-content-recovery-boundary-live-verifier.md`: A
  repo-owned live verifier now proves dashboard runtime-overview preserves the
  Go `media_asset_content_recovery` summary/card/detail, and fallback now
  synthesizes that read-only view from control-mutation audit state instead of
  dropping to `unknown`.
- `220-dashboard-media-asset-content-boundary-live-verifier.md`: A repo-owned
  live verifier now proves dashboard runtime-overview preserves the Go
  `media_asset_content` summary/card/detail and sampled recovery/preflight
  endpoints, and fallback now keeps that read-only view when direct overview
  aggregation times out.
- `221-goal-verifier-stdout-modes.md`: The unified goal verifier now writes the
  full artifact first and exposes explicit `summary/full/none` stdout modes, so
  current-turn verification no longer depends on streaming the entire artifact
  JSON to stdout.
- `222-receiver-status-cleanup-stale-live-smoke.md`: Go runtime now exposes an
  explicit stale receiver-status cleanup mutation plus a repo-owned temp-runtime
  live smoke that proves stale receiver records are removed without deleting the
  active receiver record.
- `223-dashboard-knowledge-pipelines-top-level-parity.md`: Dashboard
  runtime-overview normalize paths now preserve top-level
  `knowledge_pipelines`, so live dashboard parity checks no longer pass card
  detail while dropping the top-level payload.
- `224-agent-worker-status-startup-prune-live-smoke.md`: Go runtime now prunes
  stale `agent-worker-statuses` during repository load, and a repo-owned
  temp-runtime smoke proves stale heartbeat residue does not come back after
  restart.
- `225-agent-job-external-lease-approval-preflight.md`: Go runtime now exposes
  a read-only approval-bound preflight endpoint for `agent_job` external lease
  result-ack cutover, so callers no longer have to manually stitch together the
  plan and control-mutation approval gate.
- `226-agent-job-external-lease-launcher-preflight.md`: Repo-local launcher now
  accepts the explicit external-lease/result-ack flags needed for an isolated
  temp runtime, and a repo-owned launcher smoke proves those flags surface in
  runtime-config, queue topology, and approval-bound preflight.
- `227-agent-job-external-lease-launcher-bundle.md`: Go now exposes a read-only
  canonical launcher bundle for `agent_job` external lease result-ack cutover,
  and a repo-owned live smoke consumes that bundle to launch a temp runtime and
  verify promoted ownership plus approval-bound preflight.
- `228-agent-job-external-lease-cutover-diff.md`: Go now exposes a read-only
  diff between the current runtime and the canonical launcher bundle for
  `agent_job` external lease result-ack cutover, and a repo-owned live verifier
  proves both the blocked live-runtime drift and the promoted temp-runtime
  `zero drift -> worker coverage gate -> ready` chain.
- Agent architecture boundary specs live under `../agent-architecture/`.
