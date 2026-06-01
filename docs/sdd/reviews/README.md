# SDD Review Records

Create one review record per significant AI-generated or architecture-sensitive
change.

Suggested filename:

```text
YYYYMMDD-short-change-name.md
```

Suggested content:

```text
# Review: short change name

Spec:
Implementation summary:
Tests run:
Findings:
Decision:
Follow-ups:
```

Current architecture review sequence:

- `2026-05-30-round1-architecture-boundary-review.md`
- `2026-05-30-round2-group-memory-rag-review.md`
- `2026-05-30-round3-operational-safety-review.md`
- `2026-05-30-round4-final-design-gate-review.md`
- `2026-05-30-round5-overall-readiness-review.md`
- `2026-05-30-go-migration-sdd-design-review.md`
- `2026-05-30-phase1-contract-model-implementation.md`
- `2026-05-30-phase2-shadow-mode-implementation.md`
- `2026-05-30-phase2-1-observe-only-shadow-coverage.md`
- `2026-05-30-phase2-2-shadow-dashboard-audit.md`
- `2026-05-30-phase2-3-go-shadow-audit-jsonl.md`
- `2026-05-30-compat-channel-media-group-memory.md`
- `2026-05-30-phase4-1-go-outbox-control-plane.md`
- `2026-05-30-media-registry-sdd-design-review.md`
- `2026-05-30-phase5-1-go-media-registry-metadata.md`
- `2026-05-30-agent-job-orchestration-sdd-review.md`
- `2026-05-30-phase7-1-go-agent-job-control-plane.md`
- `2026-05-30-agent-gateway-rename-review.md`
- `2026-05-30-phase7-2-python-agent-gateway-client.md`
- `2026-05-30-phase7-3-image-job-agent-job-bridge.md`
- `2026-05-30-phase7-4-python-image-agent-worker.md`
- `2026-05-30-phase7-5-knowledge-agent-worker.md`
- `2026-05-30-phase7-6-go-agent-job-persistence.md`
- `2026-05-30-phase8-1-go-media-content-access.md`
- `2026-05-30-phase8-2-go-shadow-media-registration.md`
- `2026-05-30-phase8-3-go-media-registry-persistence.md`
- `2026-05-30-phase8-4-dashboard-media-links.md`
- `2026-05-30-phase8-5-dashboard-message-media-assets.md`
- `2026-05-30-phase8-6-send-ledger-control-plane.md`
- `2026-05-30-phase8-7-python-send-ledger-bridge.md`
- `2026-05-30-phase8-8-send-ledger-dashboard-diagnostics.md`
- `2026-05-30-phase8-9-go-outbox-persistence.md`
- `2026-05-30-phase8-10-outbox-dashboard-diagnostics.md`
- `2026-05-30-phase8-11-outbox-lease-control-plane.md`
- `2026-05-30-phase8-12-outbox-dispatch-worker.md`
- `2026-05-30-phase8-13-outbox-failure-kind.md`
- `2026-05-30-phase8-14-outbox-retryability-policy.md`
- `2026-05-30-phase8-15-go-inbox-raw-store.md`
- `2026-05-30-phase8-16-group-memory-runtime-inbox-source.md`
- `2026-05-30-phase8-17-ragflow-runtime-inbox-source.md`
- `2026-05-30-phase8-18-knowledge-checkpoints.md`
- `2026-05-30-phase8-19-runtime-json-utf8.md`
- `2026-05-30-phase8-20-knowledge-checkpoint-dashboard.md`
- `2026-05-30-phase8-21-media-safe-root-discovery.md`
- `2026-05-30-phase8-22-knowledge-worker-diagnostics.md`
- `2026-05-30-phase8-23-agent-job-event-stream.md`
- `2026-05-30-phase8-24-proactive-scheduling-state.md`
- `2026-05-30-phase8-25-python-proactive-state-runtime-bridge.md`
- `2026-05-30-phase8-26-dashboard-media-filename-fallback.md`
- `2026-05-30-phase8-27-delivery-dispatch-plan.md`
- `2026-05-30-phase8-28-telegram-delivery-adapter.md`
- `2026-05-30-phase8-29-runtime-backed-message-push.md`
- `2026-05-30-phase8-30-dashboard-media-asset-metadata-fallback.md`
- `2026-05-30-phase8-31-onebot-delivery-adapter.md`
- `2026-05-30-phase8-32-onebot-websocket-delivery.md`
- `2026-05-30-phase8-33-runtime-contract-fixtures.md`
- `2026-05-30-phase8-34-dashboard-worker-contract-smoke.md`
- `2026-05-30-phase8-35-rag-eval-agent-job-worker.md`
- `2026-05-30-phase8-36-runtime-dashboard-overview.md`
- `2026-05-30-phase8-37-rag-eval-dashboard.md`
- `2026-05-30-phase8-38-outbox-event-stream.md`
- `2026-05-30-phase8-39-external-queue-backend-design.md`
- `2026-05-30-phase8-40-nats-shadow-publish.md`
- `2026-05-30-phase8-41-queue-shadow-diagnostics.md`
- `2026-05-30-phase8-42-nats-dual-read-compare.md`
- `2026-05-30-phase8-43-external-lease-gate.md`
- `2026-05-30-phase8-44-nats-live-smoke.md`
- `2026-05-30-phase8-45-external-lease-outbox-executor.md`
- `2026-05-31-phase8-46-external-lease-nats-smoke.md`
- `2026-05-31-phase8-47-dev-mirror-source.md`
- `2026-05-31-phase8-48-agent-job-external-lease-gate.md`
- `2026-05-31-phase8-49-agent-job-lease-token.md`
- `2026-05-31-phase8-50-agent-job-lease-renew.md`
- `2026-05-31-phase8-51-agent-job-strict-token-mode.md`
- `2026-05-31-phase8-52-agent-job-lease-work.md`
- `2026-05-31-phase8-53-agent-job-timeout-recovery.md`
- `2026-05-31-phase8-54-agent-job-result-ack-mapping.md`
- `2026-05-31-phase8-55-agent-job-nats-scope-gate.md`
- `2026-05-31-phase8-56-agent-job-recovery-runner.md`
- `2026-05-31-phase8-57-agent-job-nats-flow-smoke.md`
- `2026-05-31-phase8-58-delivery-adapter-diagnostics.md`
- `2026-05-31-phase8-59-private-echo-ledger.md`
- `2026-05-31-phase8-60-runtime-adapter-dashboard.md`
- `2026-05-31-phase8-61-runtime-queue-dashboard.md`
- `2026-05-31-phase8-62-delivery-dispatch-readiness.md`
- `2026-05-31-phase8-63-outbox-worker-readiness-gate.md`
- `2026-05-31-phase8-64-outbox-dashboard-readiness.md`
- `2026-05-31-phase8-65-agent-job-dashboard-recovery.md`
- `2026-05-31-phase8-66-agent-job-metrics.md`
- `2026-05-31-phase8-67-outbox-metrics.md`
- `2026-05-31-phase8-68-inbox-metrics.md`
- `2026-05-31-phase8-69-send-ledger-metrics.md`
- `2026-05-31-phase8-70-runtime-overview-aggregate.md`
- `2026-05-31-phase8-71-go-local-outbox-worker.md`
- `2026-05-31-phase8-72-runtime-worker-diagnostics.md`
- `2026-05-31-phase8-73-delivery-adapter-health.md`
- `2026-05-31-phase8-74-runtime-overview-adapter-health.md`
- `2026-05-31-phase8-75-runtime-config-diagnostics.md`
- `2026-05-31-phase8-76-runtime-config-live-preflight.md`
- `2026-05-31-phase8-77-delivery-smoke-readiness.md`
- `2026-05-31-phase8-78-runtime-overview-delivery-smoke.md`
- `2026-05-31-phase8-79-runtime-state-defaults.md`
- `2026-05-31-phase8-80-receiver-status-heartbeat.md`
- `2026-05-31-phase8-81-receiver-lease-persistence.md`
- `2026-05-31-phase8-82-observe-capture-activity-inference.md`
- `2026-05-31-phase8-83-inbound-dedupe-runtime.md`
- `2026-05-31-phase8-84-qq-inbound-dedupe-runtime.md`
- `2026-05-31-phase8-85-qq-file-notice-dedupe.md`
- `2026-05-31-phase8-86-inbound-dedupe-metrics.md`
- `2026-05-31-phase8-87-agent-job-admission-dedupe.md`
- `2026-05-31-phase8-88-agent-worker-status.md`
- `2026-05-31-phase8-89-proactive-anyaction-quota.md`
- `2026-05-31-phase8-90-proactive-seen-rejection-state.md`
- `2026-05-31-phase8-91-proactive-retention-cleanup.md`
- `2026-05-31-phase8-92-proactive-bg-context-global-mark.md`
- `2026-05-31-phase8-93-scheduler-job-store.md`
- `2026-05-31-phase8-94-scheduler-diagnostics.md`
- `2026-05-31-phase8-95-scheduler-execution-lease.md`
- `2026-05-31-phase8-96-scheduler-job-crud.md`
- `2026-05-31-phase8-97-scheduler-completion-mutation.md`
- `2026-05-31-phase8-98-scheduler-recovery-reconciliation.md`
- `2026-05-31-phase8-99-proactive-drift-state.md`
- `2026-05-31-phase8-100-proactive-tick-log-state.md`
- `2026-05-31-phase8-101-dashboard-proactive-tick-log-runtime-fallback.md`
- `2026-05-31-phase8-102-agent-worker-status-lease-fencing.md`
- `2026-05-31-phase8-103-agent-worker-status-heartbeat-renewal.md`
- `2026-05-31-phase8-104-external-lease-execution-diagnostics.md`
- `2026-05-31-phase8-105-runtime-overview-external-lease-diagnostics.md`
- `2026-05-31-phase8-106-outbox-account-pressure-diagnostics.md`
- `2026-05-31-phase8-107-outbox-account-rate-limit.md`
- `2026-05-31-phase8-108-external-lease-outbox-account-rate-limit.md`
- `2026-05-31-phase8-109-queue-provider-capability-diagnostics.md`
- `2026-05-31-phase8-110-runtime-overview-queue-provider-capability.md`
- `2026-05-31-phase8-111-external-lease-local-worker-conflict-gate.md`
- `2026-05-31-phase8-112-queue-execution-owner-diagnostics.md`
- `2026-05-31-phase8-113-agent-job-pressure-diagnostics.md`
- `2026-05-31-phase8-114-agent-job-worker-coverage-diagnostics.md`
- `2026-05-31-phase8-115-knowledge-pipeline-diagnostics.md`
- `2026-05-31-phase8-116-knowledge-pipeline-source-lag-diagnostics.md`
- `2026-05-31-phase8-117-knowledge-pipeline-checkpoint-age-diagnostics.md`
- `2026-05-31-phase8-118-knowledge-pipeline-job-lease-freshness-diagnostics.md`
- `2026-05-31-phase8-119-knowledge-pipeline-rag-dataset-state-diagnostics.md`
- `2026-05-31-phase8-120-knowledge-pipeline-configured-rag-dataset-bindings.md`
- `2026-05-31-phase8-121-knowledge-pipeline-rag-ingest-snapshot-diagnostics.md`
- `2026-05-31-phase8-122-go-owned-knowledge-job-planner.md`
- `2026-05-31-phase8-123-knowledge-job-planner-preview.md`
- `2026-05-31-phase8-124-runtime-overview-knowledge-planner-preview.md`
- `2026-05-31-phase8-125-knowledge-job-planner-readiness.md`
- `2026-05-31-phase8-126-runtime-overview-knowledge-planner-readiness.md`
- `2026-05-31-phase8-127-knowledge-pipeline-rag-index-state.md`
- `2026-05-31-phase8-128-outbound-cutover-readiness.md`
- `2026-05-31-phase8-129-outbound-cutover-plan.md`
- `2026-05-31-phase8-130-runtime-overview-outbound-cutover-plan.md`
- `2026-05-31-phase8-131-agent-job-external-lease-readiness.md`
- `2026-05-31-phase8-132-runtime-overview-agent-job-external-lease-readiness.md`
- `2026-05-31-phase8-133-agent-job-external-lease-plan.md`
- `2026-05-31-phase8-134-runtime-overview-agent-job-external-lease-plan.md`
- `2026-05-31-phase8-135-agent-job-capacity-plan.md`
- `2026-05-31-phase8-136-runtime-overview-agent-job-capacity-plan.md`
- `2026-05-31-phase8-137-media-asset-content-diagnostics.md`
- `2026-05-31-phase8-138-runtime-overview-media-asset-content.md`
- `2026-05-31-phase8-139-runtime-overview-delivery-smoke.md`
- `2026-05-31-phase8-140-dashboard-runtime-overview-new-fields.md`
- `2026-05-31-phase8-141-knowledge-job-planner-cutover-plan.md`
- `2026-05-31-phase8-142-dashboard-knowledge-planner-cutover-plan.md`
- `2026-06-01-phase8-143-dashboard-runtime-control-plane-details.md`
- `2026-06-01-phase8-144-queue-topology-read-model.md`
- `2026-06-01-phase8-145-runtime-overview-queue-topology.md`
