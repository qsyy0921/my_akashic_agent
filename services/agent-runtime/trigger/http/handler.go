package httptrigger

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/api/dto"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	inport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/in"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/types"
)

func RegisterRoutes(
	mux *http.ServeMux,
	ingestor inport.MessageIngestor,
	shadowIngestor inport.ShadowMessageIngestor,
	shadowViewer inport.ShadowAuditViewer,
	sender inport.MessageSender,
	imageJobs inport.ImageJobManager,
	outbox inport.OutboxManager,
	mediaAssets inport.MediaAssetManager,
	agentJobs inport.AgentJobManager,
	sendLedger inport.SendLedgerManager,
	inboxEvents inport.InboxEventViewer,
) {
	mux.Handle("/healthz", HealthHandler())
	mux.Handle("/v1/inbound", IngestHandler(ingestor))
	mux.Handle("/v1/shadow/inbound", ShadowIngestHandler(shadowIngestor))
	mux.Handle("/v1/shadow/observed", ShadowObservedHandler(shadowViewer))
	mux.Handle("/v1/outbound", SendHandler(sender))
	mux.Handle("/v1/image-jobs", ImageJobsHandler(imageJobs))
	mux.Handle("/v1/image-jobs/", ImageJobStateHandler(imageJobs))
	mux.Handle("/v1/outbox", OutboxListHandler(outbox))
	mux.Handle("/v1/outbox/lease-next", OutboxLeaseNextHandler(outbox))
	mux.Handle("/v1/outbox/", OutboxStateHandler(outbox))
	mux.Handle("/v1/media-assets", MediaAssetsHandler(mediaAssets))
	mux.Handle("/v1/media-assets/", MediaAssetStateHandler(mediaAssets))
	mux.Handle("/v1/jobs", AgentJobsHandler(agentJobs))
	mux.Handle("/v1/jobs/lease-next", AgentJobLeaseNextHandler(agentJobs))
	mux.Handle("/v1/jobs/lease-work", AgentJobLeaseWorkHandler(agentJobs))
	mux.Handle("/v1/jobs/recover-expired", AgentJobRecoverExpiredHandler(agentJobs))
	mux.Handle("/v1/jobs/", AgentJobStateHandler(agentJobs))
	mux.Handle("/v1/send-ledger/records", SendLedgerRecordsHandler(sendLedger))
	mux.Handle("/v1/send-ledger/recent", SendLedgerRecentHandler(sendLedger))
	mux.Handle("/v1/send-ledger/private-echo", SendLedgerPrivateEchoHandler(sendLedger))
	mux.Handle("/v1/send-ledger/metrics", SendLedgerMetricsHandler(sendLedger))
	mux.Handle("/v1/inbox", InboxEventsHandler(inboxEvents))
	mux.Handle("/v1/inbox/", InboxEventStateHandler(inboxEvents))
}

func RegisterKnowledgeCheckpointRoutes(
	mux *http.ServeMux,
	checkpoints inport.KnowledgeCheckpointManager,
) {
	mux.Handle("/v1/knowledge-checkpoints", KnowledgeCheckpointsHandler(checkpoints))
	mux.Handle("/v1/knowledge-checkpoints/", KnowledgeCheckpointStateHandler(checkpoints))
}

func RegisterKnowledgeDiagnosticsRoutes(
	mux *http.ServeMux,
	diagnostics inport.KnowledgeWorkerDiagnosticsViewer,
) {
	mux.Handle("/v1/knowledge-worker-diagnostics", KnowledgeWorkerDiagnosticsHandler(diagnostics))
}

func RegisterKnowledgePipelineDiagnosticsRoutes(
	mux *http.ServeMux,
	diagnostics inport.KnowledgePipelineDiagnosticsViewer,
) {
	mux.Handle("/v1/knowledge-pipeline-diagnostics", KnowledgePipelineDiagnosticsHandler(diagnostics))
}

func RegisterAgentJobEventRoutes(
	mux *http.ServeMux,
	jobEvents inport.AgentJobEventViewer,
) {
	mux.Handle("/v1/job-events", AgentJobEventsHandler(jobEvents))
}

func RegisterAgentJobMetricsRoutes(
	mux *http.ServeMux,
	metrics inport.AgentJobMetricsViewer,
) {
	mux.Handle("/v1/job-metrics", AgentJobMetricsHandler(metrics))
}

func RegisterInboxMetricsRoutes(
	mux *http.ServeMux,
	metrics inport.InboxMetricsViewer,
) {
	mux.Handle("/v1/inbox-metrics", InboxMetricsHandler(metrics))
}

func RegisterInboundDedupeRoutes(
	mux *http.ServeMux,
	manager inport.InboundDedupeManager,
) {
	mux.Handle("/v1/inbound-dedupe/check", InboundDedupeCheckHandler(manager))
	mux.Handle("/v1/inbound-dedupe/records", InboundDedupeRecordsHandler(manager))
	mux.Handle("/v1/inbound-dedupe/metrics", InboundDedupeMetricsHandler(manager))
}

func RegisterOutboxEventRoutes(
	mux *http.ServeMux,
	outboxEvents inport.OutboxDeliveryEventViewer,
) {
	mux.Handle("/v1/outbox-events", OutboxDeliveryEventsHandler(outboxEvents))
}

func RegisterOutboxMetricsRoutes(
	mux *http.ServeMux,
	metrics inport.OutboxMetricsViewer,
) {
	mux.Handle("/v1/outbox-metrics", OutboxMetricsHandler(metrics))
}

func RegisterQueueBackendRoutes(
	mux *http.ServeMux,
	queueBackend inport.QueueBackendViewer,
) {
	mux.Handle("/v1/queue-backend", QueueBackendHandler(queueBackend))
}

func RegisterDeliveryDispatchRoutes(
	mux *http.ServeMux,
	planner inport.DeliveryDispatchPlanner,
) {
	mux.Handle("/v1/delivery-dispatch/plan", DeliveryDispatchPlanHandler(planner))
	mux.Handle("/v1/delivery-dispatch/readiness", DeliveryDispatchReadinessHandler(planner))
	mux.Handle("/v1/delivery-dispatch/send", DeliveryDispatchSendHandler(planner))
}

func RegisterDeliveryAdapterDiagnosticsRoutes(
	mux *http.ServeMux,
	viewer inport.DeliveryAdapterDiagnosticsViewer,
) {
	mux.Handle("/v1/delivery-adapters", DeliveryAdaptersHandler(viewer))
}

func RegisterDeliveryAdapterHealthRoutes(
	mux *http.ServeMux,
	viewer inport.DeliveryAdapterHealthViewer,
) {
	mux.Handle("/v1/delivery-adapters/health", DeliveryAdapterHealthHandler(viewer))
}

func RegisterDeliverySmokeRoutes(
	mux *http.ServeMux,
	checker inport.DeliverySmokeReadinessChecker,
) {
	mux.Handle("/v1/delivery-smoke/readiness", DeliverySmokeReadinessHandler(checker))
}

func RegisterObserveTargetRoutes(
	mux *http.ServeMux,
	manager inport.ObserveTargetManager,
) {
	mux.Handle("/v1/observe-targets/sync", ObserveTargetsSyncHandler(manager))
	mux.Handle("/v1/observe-targets", ObserveTargetsHandler(manager))
}

func RegisterObserveCaptureDiagnosticsRoutes(
	mux *http.ServeMux,
	viewer inport.ObserveCaptureDiagnosticsViewer,
) {
	mux.Handle("/v1/observe-capture-diagnostics", ObserveCaptureDiagnosticsHandler(viewer))
}

func RegisterReceiverStatusRoutes(
	mux *http.ServeMux,
	manager inport.ReceiverStatusManager,
) {
	mux.Handle("/v1/receiver-statuses/report", ReceiverStatusReportHandler(manager))
	mux.Handle("/v1/receiver-statuses", ReceiverStatusesHandler(manager))
	mux.Handle("/v1/receiver-leases/acquire", ReceiverLeaseAcquireHandler(manager))
	mux.Handle("/v1/receiver-leases/renew", ReceiverLeaseRenewHandler(manager))
	mux.Handle("/v1/receiver-leases/release", ReceiverLeaseReleaseHandler(manager))
	mux.Handle("/v1/receiver-leases", ReceiverLeasesHandler(manager))
}

func ObserveCaptureDiagnosticsHandler(viewer inport.ObserveCaptureDiagnosticsViewer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if viewer == nil {
			http.Error(w, "observe capture diagnostics disabled", http.StatusNotImplemented)
			return
		}
		view, err := viewer.GetObserveCaptureDiagnostics(r.Context(), query.ObserveCaptureDiagnosticsFilter{
			Limit: parsePositiveInt(r.URL.Query().Get("limit"), 200, 1000),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{
			Code: types.ErrorCodeOK,
			Data: view,
		})
	})
}

func KnowledgePipelineDiagnosticsHandler(viewer inport.KnowledgePipelineDiagnosticsViewer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if viewer == nil {
			http.Error(w, "knowledge pipeline diagnostics disabled", http.StatusNotImplemented)
			return
		}
		view, err := viewer.GetKnowledgePipelineDiagnostics(r.Context(), query.KnowledgePipelineDiagnosticsFilter{
			Limit:             parsePositiveInt(r.URL.Query().Get("limit"), 50, 200),
			StaleAfterSeconds: parsePositiveInt(r.URL.Query().Get("stale_after_seconds"), 900, 24*60*60),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{
			Code: types.ErrorCodeOK,
			Data: view,
		})
	})
}

func RegisterRuntimeOverviewRoutes(
	mux *http.ServeMux,
	viewer inport.RuntimeOverviewViewer,
) {
	mux.Handle("/v1/runtime-overview", RuntimeOverviewHandler(viewer))
}

func RegisterRuntimeConfigRoutes(
	mux *http.ServeMux,
	viewer inport.RuntimeConfigViewer,
) {
	mux.Handle("/v1/runtime-config", RuntimeConfigHandler(viewer))
}

func RegisterRuntimeWorkerDiagnosticsRoutes(
	mux *http.ServeMux,
	viewer inport.RuntimeWorkerDiagnosticsViewer,
) {
	mux.Handle("/v1/runtime-workers", RuntimeWorkerDiagnosticsHandler(viewer))
}

func RegisterAgentWorkerStatusRoutes(
	mux *http.ServeMux,
	manager inport.AgentWorkerStatusManager,
) {
	mux.Handle("/v1/agent-worker-statuses/report", AgentWorkerStatusReportHandler(manager))
	mux.Handle("/v1/agent-worker-statuses", AgentWorkerStatusesHandler(manager))
}

func RegisterProactiveStateRoutes(
	mux *http.ServeMux,
	proactiveState inport.ProactiveStateManager,
) {
	mux.Handle("/v1/proactive/deliveries", ProactiveDeliveriesHandler(proactiveState))
	mux.Handle("/v1/proactive/deliveries/duplicate", ProactiveDeliveryDuplicateHandler(proactiveState))
	mux.Handle("/v1/proactive/deliveries/count", ProactiveDeliveryCountHandler(proactiveState))
	mux.Handle("/v1/proactive/seen-items", ProactiveSeenItemsHandler(proactiveState))
	mux.Handle("/v1/proactive/seen-items/seen", ProactiveSeenItemsSeenHandler(proactiveState))
	mux.Handle("/v1/proactive/rejection-cooldowns", ProactiveRejectionCooldownsHandler(proactiveState))
	mux.Handle("/v1/proactive/rejection-cooldowns/cooled", ProactiveRejectionCooldownsCooledHandler(proactiveState))
	mux.Handle("/v1/proactive/context-only", ProactiveContextOnlyHandler(proactiveState))
	mux.Handle("/v1/proactive/context-only/last", ProactiveContextOnlyLastHandler(proactiveState))
	mux.Handle("/v1/proactive/context-only/count", ProactiveContextOnlyCountHandler(proactiveState))
	mux.Handle("/v1/proactive/drift-runs", ProactiveDriftRunsHandler(proactiveState))
	mux.Handle("/v1/proactive/drift-runs/last", ProactiveDriftRunLastHandler(proactiveState))
	mux.Handle("/v1/proactive/drift/finish", ProactiveDriftFinishHandler(proactiveState))
	mux.Handle("/v1/proactive/drift/summary", ProactiveDriftSummaryHandler(proactiveState))
	mux.Handle("/v1/proactive/drift/skills/", ProactiveDriftSkillStateHandler(proactiveState))
	mux.Handle("/v1/proactive/tick-logs", ProactiveTickLogsHandler(proactiveState))
	mux.Handle("/v1/proactive/tick-logs/start", ProactiveTickLogStartHandler(proactiveState))
	mux.Handle("/v1/proactive/tick-logs/finish", ProactiveTickLogFinishHandler(proactiveState))
	mux.Handle("/v1/proactive/tick-steps", ProactiveTickStepLogHandler(proactiveState))
	mux.Handle("/v1/proactive/tick-logs/", ProactiveTickLogDetailHandler(proactiveState))
	mux.Handle("/v1/proactive/bg-context/main", ProactiveBGContextMainHandler(proactiveState))
	mux.Handle("/v1/proactive/bg-context/main/last", ProactiveBGContextMainLastHandler(proactiveState))
	mux.Handle("/v1/proactive/anyaction/quota", ProactiveAnyActionQuotaHandler(proactiveState))
	mux.Handle("/v1/proactive/anyaction/actions", ProactiveAnyActionRecordHandler(proactiveState))
	mux.Handle("/v1/proactive/cleanup", ProactiveCleanupHandler(proactiveState))
}

func RegisterSchedulerJobRoutes(
	mux *http.ServeMux,
	schedulerJobs inport.SchedulerJobManager,
) {
	mux.Handle("/v1/scheduler/jobs", SchedulerJobsHandler(schedulerJobs))
	mux.Handle("/v1/scheduler/jobs/snapshot", SchedulerJobSnapshotHandler(schedulerJobs))
	mux.Handle("/v1/scheduler/jobs/upsert", SchedulerJobUpsertHandler(schedulerJobs))
	mux.Handle("/v1/scheduler/jobs/", SchedulerJobStateHandler(schedulerJobs))
	mux.Handle("/v1/scheduler/leases/acquire", SchedulerExecutionLeaseAcquireHandler(schedulerJobs))
	mux.Handle("/v1/scheduler/leases/renew", SchedulerExecutionLeaseRenewHandler(schedulerJobs))
	mux.Handle("/v1/scheduler/leases/release", SchedulerExecutionLeaseReleaseHandler(schedulerJobs))
	mux.Handle("/v1/scheduler/leases", SchedulerExecutionLeasesHandler(schedulerJobs))
	if diagnostics, ok := schedulerJobs.(inport.SchedulerJobDiagnosticsViewer); ok {
		mux.Handle("/v1/scheduler/diagnostics", SchedulerJobDiagnosticsHandler(diagnostics))
	}
}

func HealthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: map[string]string{"status": "ok"}})
	})
}

func IngestHandler(ingestor inport.MessageIngestor) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request dto.IngestMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}

		cmd, err := toIngestCommand(request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := ingestor.Ingest(r.Context(), cmd); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, http.StatusAccepted, types.Result{Code: types.ErrorCodeOK})
	})
}

func ShadowObservedHandler(shadowViewer inport.ShadowAuditViewer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		limit := parsePositiveInt(r.URL.Query().Get("limit"), 50, 200)
		items, err := shadowViewer.ListObserved(r.Context(), limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{
			Code: types.ErrorCodeOK,
			Data: items,
		})
	})
}

func ShadowIngestHandler(shadowIngestor inport.ShadowMessageIngestor) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request dto.IngestMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}

		cmd, err := toIngestCommand(request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		decision, err := shadowIngestor.ShadowIngest(r.Context(), cmd)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, http.StatusAccepted, types.Result{
			Code: types.ErrorCodeOK,
			Data: dto.ShadowIngestResponse{
				Decision: dto.LoopDecisionDTO{
					Action: string(decision.Action),
					Reason: decision.Reason,
				},
			},
		})
	})
}

func InboxEventsHandler(inboxEvents inport.InboxEventViewer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if inboxEvents == nil {
			http.Error(w, "inbox event viewer disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		channelKind := r.URL.Query().Get("channel_kind")
		if channelKind == "" {
			channelKind = r.URL.Query().Get("platform")
		}
		items, err := inboxEvents.ListInboxEvents(r.Context(), query.InboxEventFilter{
			Limit:            parsePositiveInt(r.URL.Query().Get("limit"), 50, 5000),
			AfterSeq:         parseNonNegativeInt(r.URL.Query().Get("after_seq"), -1),
			AfterSeqSet:      strings.TrimSpace(r.URL.Query().Get("after_seq")) != "",
			Order:            strings.ToLower(strings.TrimSpace(r.URL.Query().Get("order"))),
			ChannelKind:      channelKind,
			AccountID:        r.URL.Query().Get("account_id"),
			ConversationID:   r.URL.Query().Get("conversation_id"),
			ConversationType: r.URL.Query().Get("conversation_type"),
			SenderID:         r.URL.Query().Get("sender_id"),
			DecisionAction:   r.URL.Query().Get("decision_action"),
			ObserveOnly:      r.URL.Query().Get("observe_only"),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: items})
	})
}

func InboxEventStateHandler(inboxEvents inport.InboxEventViewer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if inboxEvents == nil {
			http.Error(w, "inbox event viewer disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		eventID := strings.TrimPrefix(r.URL.Path, "/v1/inbox/")
		eventID, err := url.PathUnescape(eventID)
		if err != nil || strings.TrimSpace(eventID) == "" {
			http.Error(w, "missing inbox event id", http.StatusBadRequest)
			return
		}
		item, err := inboxEvents.GetInboxEvent(r.Context(), eventID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: item})
	})
}

func KnowledgeCheckpointStateHandler(checkpoints inport.KnowledgeCheckpointManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if checkpoints == nil {
			http.Error(w, "knowledge checkpoint manager disabled", http.StatusNotImplemented)
			return
		}
		checkpointID := strings.TrimPrefix(r.URL.Path, "/v1/knowledge-checkpoints/")
		checkpointID, err := url.PathUnescape(checkpointID)
		if err != nil || strings.TrimSpace(checkpointID) == "" {
			http.Error(w, "missing checkpoint id", http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodGet:
			checkpoint, err := checkpoints.Get(r.Context(), checkpointID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: checkpoint})
		case http.MethodPost, http.MethodPut:
			var request dto.UpsertKnowledgeCheckpointRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, "invalid json body", http.StatusBadRequest)
				return
			}
			timestamp, err := parseOptionalTimestamp(request.Timestamp)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			checkpoint, err := checkpoints.Upsert(r.Context(), command.UpsertKnowledgeCheckpointCommand{
				CheckpointID: checkpointID,
				Cursor:       request.Cursor,
				Metadata:     request.Metadata,
				Timestamp:    timestamp,
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: checkpoint})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func KnowledgeCheckpointsHandler(checkpoints inport.KnowledgeCheckpointManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if checkpoints == nil {
			http.Error(w, "knowledge checkpoint manager disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		items, err := checkpoints.List(r.Context(), query.KnowledgeCheckpointFilter{
			Limit:  parsePositiveInt(r.URL.Query().Get("limit"), 50, 200),
			Prefix: r.URL.Query().Get("prefix"),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: items})
	})
}

func SchedulerJobsHandler(schedulerJobs inport.SchedulerJobManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if schedulerJobs == nil {
			http.Error(w, "scheduler job manager disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		items, err := schedulerJobs.ListSchedulerJobs(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: items})
	})
}

func SchedulerJobSnapshotHandler(schedulerJobs inport.SchedulerJobManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if schedulerJobs == nil {
			http.Error(w, "scheduler job manager disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost && r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.SchedulerJobSnapshotRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		cmd, err := toReplaceSchedulerJobsCommand(request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		view, err := schedulerJobs.ReplaceSchedulerJobs(r.Context(), cmd)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusAccepted, types.Result{Code: types.ErrorCodeOK, Data: view})
	})
}

func SchedulerJobUpsertHandler(schedulerJobs inport.SchedulerJobManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if schedulerJobs == nil {
			http.Error(w, "scheduler job manager disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost && r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.SchedulerJobUpsertRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		job, err := toSchedulerJobCommand(request.Job)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		view, err := schedulerJobs.UpsertSchedulerJob(r.Context(), command.UpsertSchedulerJobCommand{
			Job:    job,
			Source: request.Source,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: view})
	})
}

func SchedulerJobStateHandler(schedulerJobs inport.SchedulerJobManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if schedulerJobs == nil {
			http.Error(w, "scheduler job manager disabled", http.StatusNotImplemented)
			return
		}
		rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/scheduler/jobs/"), "/")
		if r.Method == http.MethodPost && strings.HasSuffix(rest, "/complete") {
			completeJobID := strings.TrimSuffix(rest, "/complete")
			completeJobID = strings.TrimSuffix(completeJobID, "/")
			if completeJobID == "" || strings.Contains(completeJobID, "/") {
				http.Error(w, "invalid scheduler job id", http.StatusBadRequest)
				return
			}
			handleSchedulerJobComplete(w, r, schedulerJobs, completeJobID)
			return
		}
		jobID := rest
		if jobID == "" || strings.Contains(jobID, "/") {
			http.Error(w, "invalid scheduler job id", http.StatusBadRequest)
			return
		}
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		view, err := schedulerJobs.DeleteSchedulerJob(r.Context(), command.DeleteSchedulerJobCommand{
			ID:     jobID,
			Source: r.URL.Query().Get("source"),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: view})
	})
}

func handleSchedulerJobComplete(w http.ResponseWriter, r *http.Request, schedulerJobs inport.SchedulerJobManager, jobID string) {
	var request dto.SchedulerJobCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	timestamp, err := parseOptionalTimestamp(request.Timestamp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var job command.SchedulerJobCommand
	if strings.TrimSpace(request.Action) == "reschedule" {
		job, err = toSchedulerJobCommand(request.Job)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	view, err := schedulerJobs.CompleteSchedulerJob(r.Context(), command.CompleteSchedulerJobCommand{
		ID:         jobID,
		Source:     request.Source,
		HolderID:   request.HolderID,
		LeaseToken: request.LeaseToken,
		Action:     request.Action,
		Job:        job,
		Timestamp:  timestamp,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: view})
}

func SchedulerJobDiagnosticsHandler(viewer inport.SchedulerJobDiagnosticsViewer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if viewer == nil {
			http.Error(w, "scheduler diagnostics disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		item, err := viewer.GetSchedulerJobDiagnostics(r.Context(), query.SchedulerJobDiagnosticsFilter{
			Limit:          parsePositiveInt(r.URL.Query().Get("limit"), 50, 200),
			Timestamp:      r.URL.Query().Get("timestamp"),
			DueSoonSeconds: parsePositiveInt(r.URL.Query().Get("due_soon_seconds"), 300, 24*60*60),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: item})
	})
}

func SchedulerExecutionLeasesHandler(schedulerJobs inport.SchedulerJobManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if schedulerJobs == nil {
			http.Error(w, "scheduler job manager disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		view, err := schedulerJobs.ListSchedulerExecutionLeases(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: view})
	})
}

func SchedulerExecutionLeaseAcquireHandler(schedulerJobs inport.SchedulerJobManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if schedulerJobs == nil {
			http.Error(w, "scheduler job manager disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.AcquireSchedulerExecutionLeaseRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		timestamp, err := parseOptionalTimestamp(request.Timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		view, err := schedulerJobs.AcquireSchedulerExecutionLease(r.Context(), command.AcquireSchedulerExecutionLeaseCommand{
			JobID:      request.JobID,
			HolderID:   request.HolderID,
			TTLSeconds: request.TTLSeconds,
			Metadata:   request.Metadata,
			Timestamp:  timestamp,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: view})
	})
}

func SchedulerExecutionLeaseRenewHandler(schedulerJobs inport.SchedulerJobManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if schedulerJobs == nil {
			http.Error(w, "scheduler job manager disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.RenewSchedulerExecutionLeaseRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		timestamp, err := parseOptionalTimestamp(request.Timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		view, err := schedulerJobs.RenewSchedulerExecutionLease(r.Context(), command.RenewSchedulerExecutionLeaseCommand{
			JobID:      request.JobID,
			HolderID:   request.HolderID,
			LeaseToken: request.LeaseToken,
			TTLSeconds: request.TTLSeconds,
			Timestamp:  timestamp,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: view})
	})
}

func SchedulerExecutionLeaseReleaseHandler(schedulerJobs inport.SchedulerJobManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if schedulerJobs == nil {
			http.Error(w, "scheduler job manager disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.ReleaseSchedulerExecutionLeaseRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		timestamp, err := parseOptionalTimestamp(request.Timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		view, err := schedulerJobs.ReleaseSchedulerExecutionLease(r.Context(), command.ReleaseSchedulerExecutionLeaseCommand{
			JobID:      request.JobID,
			HolderID:   request.HolderID,
			LeaseToken: request.LeaseToken,
			Timestamp:  timestamp,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: view})
	})
}

func parsePositiveInt(value string, fallback int, maxValue int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	if parsed > maxValue {
		return maxValue
	}
	return parsed
}

func parseNonNegativeInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func parseBoolQuery(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func DeliveryAdaptersHandler(viewer inport.DeliveryAdapterDiagnosticsViewer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if viewer == nil {
			http.Error(w, "delivery adapter diagnostics disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		items, err := viewer.ListDeliveryAdapters(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: items})
	})
}

func DeliveryAdapterHealthHandler(viewer inport.DeliveryAdapterHealthViewer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if viewer == nil {
			http.Error(w, "delivery adapter health disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		item, err := viewer.CheckDeliveryAdapters(r.Context(), query.DeliveryAdapterHealthFilter{
			TimeoutSeconds: parsePositiveInt(r.URL.Query().Get("timeout_seconds"), 3, 30),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: item})
	})
}

func RuntimeConfigHandler(viewer inport.RuntimeConfigViewer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if viewer == nil {
			http.Error(w, "runtime config diagnostics disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		item, err := viewer.GetRuntimeConfig(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: item})
	})
}

func SendHandler(sender inport.MessageSender) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request dto.SendMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}

		cmd, err := toSendCommand(request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := sender.Send(r.Context(), cmd); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, http.StatusAccepted, types.Result{Code: types.ErrorCodeOK})
	})
}

func SendLedgerRecordsHandler(sendLedger inport.SendLedgerManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if sendLedger == nil {
			http.Error(w, "send ledger disabled", http.StatusNotImplemented)
			return
		}

		switch r.Method {
		case http.MethodGet:
			items, err := sendLedger.List(r.Context(), query.SendRecordFilter{
				Limit:          parsePositiveInt(r.URL.Query().Get("limit"), 50, 200),
				FromBotID:      r.URL.Query().Get("from_bot_id"),
				ConversationID: r.URL.Query().Get("conversation_id"),
				ContentHash:    r.URL.Query().Get("content_hash"),
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: items})
		case http.MethodPost:
			var request dto.RecordSendRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, "invalid json body", http.StatusBadRequest)
				return
			}
			timestamp, err := parseOptionalTimestamp(request.Timestamp)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			record, err := sendLedger.Record(r.Context(), command.RecordSendCommand{
				FromBotID:      request.FromBotID,
				ConversationID: request.ConversationID,
				Content:        request.Content,
				ContentHash:    request.ContentHash,
				Timestamp:      timestamp,
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			writeJSON(w, http.StatusAccepted, types.Result{Code: types.ErrorCodeOK, Data: record})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func SendLedgerRecentHandler(sendLedger inport.SendLedgerManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if sendLedger == nil {
			http.Error(w, "send ledger disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		windowSeconds := parsePositiveInt(r.URL.Query().Get("window_seconds"), 15, 86400)
		recent, err := sendLedger.RecentlySent(r.Context(), command.CheckRecentSendCommand{
			FromBotID:      r.URL.Query().Get("from_bot_id"),
			ConversationID: r.URL.Query().Get("conversation_id"),
			Content:        r.URL.Query().Get("content"),
			ContentHash:    r.URL.Query().Get("content_hash"),
			Window:         time.Duration(windowSeconds) * time.Second,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: recent})
	})
}

func SendLedgerPrivateEchoHandler(sendLedger inport.SendLedgerManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if sendLedger == nil {
			http.Error(w, "send ledger disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		windowSeconds := parsePositiveInt(r.URL.Query().Get("window_seconds"), 180, 86400)
		echo, err := sendLedger.CheckPrivateEcho(r.Context(), command.CheckPrivateEchoCommand{
			FromUserID:  r.URL.Query().Get("from_user_id"),
			ToBotID:     r.URL.Query().Get("to_bot_id"),
			Text:        r.URL.Query().Get("text"),
			HasImage:    parseBoolQuery(r.URL.Query().Get("has_image")),
			HasFile:     parseBoolQuery(r.URL.Query().Get("has_file")),
			HasForward:  parseBoolQuery(r.URL.Query().Get("has_forward")),
			ContentHash: r.URL.Query().Get("content_hash"),
			Window:      time.Duration(windowSeconds) * time.Second,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: echo})
	})
}

func SendLedgerMetricsHandler(sendLedger inport.SendLedgerManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if sendLedger == nil {
			http.Error(w, "send ledger disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		item, err := sendLedger.Metrics(r.Context(), query.SendLedgerMetricsFilter{
			Limit:          parsePositiveInt(r.URL.Query().Get("limit"), 200, 200),
			FromBotID:      r.URL.Query().Get("from_bot_id"),
			ConversationID: r.URL.Query().Get("conversation_id"),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: item})
	})
}

func OutboxListHandler(outbox inport.OutboxManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		limit := parsePositiveInt(r.URL.Query().Get("limit"), 50, 200)
		items, err := outbox.List(r.Context(), limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{
			Code: types.ErrorCodeOK,
			Data: items,
		})
	})
}

func OutboxLeaseNextHandler(outbox inport.OutboxManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.OutboxLeaseRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		timestamp, err := parseOptionalTimestamp(request.Timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		item, err := outbox.LeaseNext(r.Context(), command.LeaseNextOutboxCommand{
			WorkerID:   request.WorkerID,
			TTLSeconds: request.TTLSeconds,
			Timestamp:  timestamp,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: item})
	})
}

func OutboxStateHandler(outbox inport.OutboxManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		eventID, action := parseOutboxPath(r.URL.Path)
		if eventID == "" {
			http.Error(w, "outbox event id required", http.StatusBadRequest)
			return
		}
		if r.Method == http.MethodGet && action == "" {
			item, err := outbox.Get(r.Context(), eventID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			writeJSON(w, http.StatusOK, types.Result{
				Code: types.ErrorCodeOK,
				Data: item,
			})
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request dto.OutboxStateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		timestamp, err := parseOptionalTimestamp(request.Timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var item query.OutboxDeliveryView
		switch action {
		case "dispatching":
			item, err = outbox.MarkDispatching(r.Context(), command.MarkOutboxDispatchingCommand{
				EventID:   eventID,
				Timestamp: timestamp,
			})
		case "succeeded":
			item, err = outbox.MarkSucceeded(r.Context(), command.MarkOutboxSucceededCommand{
				EventID:   eventID,
				Timestamp: timestamp,
			})
		case "failed":
			item, err = outbox.MarkFailed(r.Context(), command.MarkOutboxFailedCommand{
				EventID:      eventID,
				ErrorKind:    request.ErrorKind,
				ErrorMessage: request.ErrorMessage,
				Timestamp:    timestamp,
			})
		case "retry":
			item, err = outbox.Retry(r.Context(), command.RetryOutboxCommand{
				EventID:   eventID,
				Timestamp: timestamp,
			})
		default:
			http.Error(w, "unknown outbox action", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{
			Code: types.ErrorCodeOK,
			Data: item,
		})
	})
}

func DeliveryDispatchPlanHandler(planner inport.DeliveryDispatchPlanner) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if planner == nil {
			http.Error(w, "delivery dispatch planner disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request dto.PlanDeliveryDispatchRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		plan, err := planner.Plan(r.Context(), command.PlanDeliveryDispatchCommand{
			EventID:          request.EventID,
			ChannelByAccount: request.ChannelByAccount,
		})
		if err != nil {
			kind := deliveryDispatchErrorKind(err)
			writeJSON(w, http.StatusBadRequest, types.Result{
				Code:    types.ErrorCodeInvalidArgument,
				Message: err.Error(),
				Data: map[string]string{
					"error_kind": kind,
				},
			})
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: plan})
	})
}

func DeliveryDispatchSendHandler(planner inport.DeliveryDispatchPlanner) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if planner == nil {
			http.Error(w, "delivery dispatch planner disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request dto.DispatchDeliveryRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		result, err := planner.Dispatch(r.Context(), command.DispatchDeliveryCommand{
			EventID:          request.EventID,
			ChannelByAccount: request.ChannelByAccount,
		})
		if err != nil {
			writeDeliveryDispatchError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: result})
	})
}

func DeliveryDispatchReadinessHandler(planner inport.DeliveryDispatchPlanner) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if planner == nil {
			http.Error(w, "delivery dispatch planner disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request dto.PlanDeliveryDispatchRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		result, err := planner.Readiness(r.Context(), command.CheckDeliveryDispatchReadinessCommand{
			EventID:          request.EventID,
			ChannelByAccount: request.ChannelByAccount,
		})
		if err != nil {
			kind := deliveryDispatchErrorKind(err)
			writeJSON(w, http.StatusBadRequest, types.Result{
				Code:    types.ErrorCodeInvalidArgument,
				Message: err.Error(),
				Data: map[string]string{
					"error_kind": kind,
				},
			})
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: result})
	})
}

func DeliverySmokeReadinessHandler(checker inport.DeliverySmokeReadinessChecker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if checker == nil {
			http.Error(w, "delivery smoke readiness disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request dto.DeliverySmokeReadinessRequest
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
		if err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(string(raw)) != "" {
			if err := json.Unmarshal(raw, &request); err != nil {
				http.Error(w, "invalid json body", http.StatusBadRequest)
				return
			}
		}
		result, err := checker.CheckDeliverySmokeReadiness(r.Context(), command.CheckDeliverySmokeReadinessCommand{
			Cases:                 deliverySmokeCaseCommands(request.Cases),
			GroupIDs:              request.GroupIDs,
			ChannelByAccount:      request.ChannelByAccount,
			IncludeSyntheticMedia: request.IncludeSyntheticMedia,
		})
		if err != nil {
			writeJSON(w, http.StatusBadRequest, types.Result{
				Code:    types.ErrorCodeInvalidArgument,
				Message: err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: result})
	})
}

func ObserveTargetsHandler(manager inport.ObserveTargetManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if manager == nil {
			http.Error(w, "observe target manager disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		view, err := manager.ListObserveTargets(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: view})
	})
}

func ObserveTargetsSyncHandler(manager inport.ObserveTargetManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if manager == nil {
			http.Error(w, "observe target manager disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost && r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.SyncObserveTargetsRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		view, err := manager.SyncObserveTargets(r.Context(), command.SyncObserveTargetsCommand{
			Source:  request.Source,
			Targets: observeTargetCommands(request.Targets),
		})
		if err != nil {
			writeJSON(w, http.StatusBadRequest, types.Result{
				Code:    types.ErrorCodeInvalidArgument,
				Message: err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: view})
	})
}

func ReceiverStatusesHandler(manager inport.ReceiverStatusManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if manager == nil {
			http.Error(w, "receiver status manager disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		view, err := manager.ListReceiverStatuses(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: view})
	})
}

func ReceiverStatusReportHandler(manager inport.ReceiverStatusManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if manager == nil {
			http.Error(w, "receiver status manager disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost && r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.ReceiverStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		timestamp, err := parseOptionalTimestamp(request.Timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		view, err := manager.ReportReceiverStatus(r.Context(), command.ReportReceiverStatusCommand{
			ReceiverID:  request.ReceiverID,
			Kind:        request.Kind,
			ChannelName: request.ChannelName,
			AccountID:   request.AccountID,
			Endpoint:    request.Endpoint,
			Status:      request.Status,
			Reason:      request.Reason,
			LastError:   request.LastError,
			Source:      request.Source,
			Metadata:    request.Metadata,
			Timestamp:   timestamp,
		})
		if err != nil {
			writeJSON(w, http.StatusBadRequest, types.Result{
				Code:    types.ErrorCodeInvalidArgument,
				Message: err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: view})
	})
}

func ReceiverLeasesHandler(manager inport.ReceiverStatusManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if manager == nil {
			http.Error(w, "receiver lease manager disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		view, err := manager.ListReceiverLeases(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: view})
	})
}

func ReceiverLeaseAcquireHandler(manager inport.ReceiverStatusManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if manager == nil {
			http.Error(w, "receiver lease manager disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost && r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.AcquireReceiverLeaseRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		timestamp, err := parseOptionalTimestamp(request.Timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		view, err := manager.AcquireReceiverLease(r.Context(), command.AcquireReceiverLeaseCommand{
			ReceiverID:  request.ReceiverID,
			Kind:        request.Kind,
			ChannelName: request.ChannelName,
			AccountID:   request.AccountID,
			HolderID:    request.HolderID,
			TTLSeconds:  request.TTLSeconds,
			Metadata:    request.Metadata,
			Timestamp:   timestamp,
		})
		if err != nil {
			writeJSON(w, http.StatusBadRequest, types.Result{Code: types.ErrorCodeInvalidArgument, Message: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: view})
	})
}

func ReceiverLeaseRenewHandler(manager inport.ReceiverStatusManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if manager == nil {
			http.Error(w, "receiver lease manager disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost && r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.RenewReceiverLeaseRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		timestamp, err := parseOptionalTimestamp(request.Timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		view, err := manager.RenewReceiverLease(r.Context(), command.RenewReceiverLeaseCommand{
			ReceiverID: request.ReceiverID,
			HolderID:   request.HolderID,
			LeaseToken: request.LeaseToken,
			TTLSeconds: request.TTLSeconds,
			Timestamp:  timestamp,
		})
		if err != nil {
			writeJSON(w, http.StatusBadRequest, types.Result{Code: types.ErrorCodeInvalidArgument, Message: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: view})
	})
}

func ReceiverLeaseReleaseHandler(manager inport.ReceiverStatusManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if manager == nil {
			http.Error(w, "receiver lease manager disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost && r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.ReleaseReceiverLeaseRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		timestamp, err := parseOptionalTimestamp(request.Timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		view, err := manager.ReleaseReceiverLease(r.Context(), command.ReleaseReceiverLeaseCommand{
			ReceiverID: request.ReceiverID,
			HolderID:   request.HolderID,
			LeaseToken: request.LeaseToken,
			Timestamp:  timestamp,
		})
		if err != nil {
			writeJSON(w, http.StatusBadRequest, types.Result{Code: types.ErrorCodeInvalidArgument, Message: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: view})
	})
}

func observeTargetCommands(items []dto.ObserveTargetRequest) []command.ObserveTargetCommand {
	commands := make([]command.ObserveTargetCommand, 0, len(items))
	for _, item := range items {
		commands = append(commands, command.ObserveTargetCommand{
			TargetID: item.TargetID,
			Channel: command.ChannelCommand{
				Kind:             item.Channel.RoutePlatform(),
				AccountID:        item.Channel.AccountID,
				ConversationID:   item.Channel.ConversationID,
				ConversationType: item.Channel.ConversationType,
			},
			ObserveOnly:  item.ObserveOnly,
			ReplyAllowed: item.ReplyAllowed,
			RequireAt:    item.RequireAt,
			AllowFrom:    item.AllowFrom,
			Enabled:      item.Enabled,
			Source:       item.Source,
			Metadata:     item.Metadata,
		})
	}
	return commands
}

func deliverySmokeCaseCommands(items []dto.DeliverySmokeCaseRequest) []command.DeliverySmokeCaseCommand {
	commands := make([]command.DeliverySmokeCaseCommand, 0, len(items))
	for _, item := range items {
		commands = append(commands, command.DeliverySmokeCaseCommand{
			Name:             item.Name,
			ChannelKind:      item.ChannelKind,
			AccountID:        item.AccountID,
			ConversationID:   item.ConversationID,
			ConversationType: item.ConversationType,
			Content:          item.Content,
			Attachments:      deliverySmokeAttachmentCommands(item.Attachments),
			Metadata:         item.Metadata,
		})
	}
	return commands
}

func deliverySmokeAttachmentCommands(items []dto.DeliverySmokeAttachmentRequest) []command.DeliverySmokeAttachmentCommand {
	commands := make([]command.DeliverySmokeAttachmentCommand, 0, len(items))
	for _, item := range items {
		commands = append(commands, command.DeliverySmokeAttachmentCommand{
			Kind:     item.Kind,
			URL:      item.URL,
			Name:     item.Name,
			MimeType: item.MimeType,
		})
	}
	return commands
}

func deliveryDispatchErrorKind(err error) string {
	if err == nil {
		return ""
	}
	if kinded, ok := err.(interface{ DeliveryErrorKind() string }); ok {
		return strings.TrimSpace(kinded.DeliveryErrorKind())
	}
	return "unknown"
}

func writeDeliveryDispatchError(w http.ResponseWriter, err error) {
	kind := deliveryDispatchErrorKind(err)
	writeJSON(w, deliveryDispatchErrorStatus(kind), types.Result{
		Code:    types.ErrorCodeInvalidArgument,
		Message: err.Error(),
		Data: map[string]string{
			"error_kind": kind,
		},
	})
}

func deliveryDispatchErrorStatus(kind string) int {
	switch kind {
	case "sender_unavailable":
		return http.StatusNotImplemented
	case "platform_timeout":
		return http.StatusGatewayTimeout
	case "platform_error", "unknown":
		return http.StatusBadGateway
	default:
		return http.StatusBadRequest
	}
}

func parseOutboxPath(path string) (string, string) {
	rest := strings.TrimPrefix(path, "/v1/outbox/")
	rest = strings.Trim(rest, "/")
	if rest == "" {
		return "", ""
	}
	parts := strings.Split(rest, "/")
	eventID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	return eventID, action
}

func MediaAssetsHandler(mediaAssets inport.MediaAssetManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := mediaAssets.List(r.Context(), mediaAssetFilterFromQuery(r))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			writeJSON(w, http.StatusOK, types.Result{
				Code: types.ErrorCodeOK,
				Data: items,
			})
		case http.MethodPost:
			var request dto.RegisterMediaAssetRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, "invalid json body", http.StatusBadRequest)
				return
			}
			cmd, err := toRegisterMediaAssetCommand(request)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			asset, err := mediaAssets.Register(r.Context(), cmd)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			writeJSON(w, http.StatusAccepted, types.Result{
				Code: types.ErrorCodeOK,
				Data: asset,
			})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func mediaAssetFilterFromQuery(r *http.Request) query.MediaAssetFilter {
	values := r.URL.Query()
	return query.MediaAssetFilter{
		Limit:                 parsePositiveInt(values.Get("limit"), 50, 200),
		ChannelKind:           firstQueryValue(values.Get("channel_kind"), values.Get("kind")),
		AccountID:             values.Get("account_id"),
		ConversationID:        values.Get("conversation_id"),
		ConversationType:      values.Get("conversation_type"),
		SourceMessageID:       values.Get("source_message_id"),
		SourceMessageIDSuffix: values.Get("source_message_id_suffix"),
		Kind:                  values.Get("asset_kind"),
	}
}

func firstQueryValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func MediaAssetStateHandler(mediaAssets inport.MediaAssetManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assetID, action := parseMediaAssetPath(r.URL.Path)
		if assetID == "" {
			http.Error(w, "media asset id required", http.StatusBadRequest)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if action == "content" {
			writeMediaAssetContent(w, r, mediaAssets, assetID)
			return
		}
		if action != "" {
			http.Error(w, "unknown media asset action", http.StatusNotFound)
			return
		}
		asset, err := mediaAssets.Get(r.Context(), assetID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{
			Code: types.ErrorCodeOK,
			Data: asset,
		})
	})
}

func writeMediaAssetContent(
	w http.ResponseWriter,
	r *http.Request,
	mediaAssets inport.MediaAssetManager,
	assetID string,
) {
	content, err := mediaAssets.OpenContent(r.Context(), assetID)
	if err != nil {
		switch {
		case errors.Is(err, outport.ErrMediaAssetContentDisabled):
			http.Error(w, err.Error(), http.StatusNotImplemented)
		case errors.Is(err, outport.ErrMediaAssetContentForbidden):
			http.Error(w, err.Error(), http.StatusForbidden)
		case errors.Is(err, outport.ErrMediaAssetContentUnavailable):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusNotFound)
		}
		return
	}
	defer content.Body.Close()
	if content.MimeType != "" {
		w.Header().Set("Content-Type", content.MimeType)
	}
	if content.SizeBytes >= 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(content.SizeBytes, 10))
	}
	if content.Name != "" {
		w.Header().Set(
			"Content-Disposition",
			mime.FormatMediaType("inline", map[string]string{"filename": content.Name}),
		)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, content.Body)
}

func parseMediaAssetPath(path string) (string, string) {
	rest := strings.TrimPrefix(path, "/v1/media-assets/")
	rest = strings.Trim(rest, "/")
	if rest == "" {
		return "", ""
	}
	parts := strings.Split(rest, "/")
	assetID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	return assetID, action
}

func AgentJobsHandler(agentJobs inport.AgentJobManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := agentJobs.List(r.Context(), query.AgentJobFilter{
				JobType: r.URL.Query().Get("type"),
				Status:  r.URL.Query().Get("status"),
				Limit:   parsePositiveInt(r.URL.Query().Get("limit"), 50, 200),
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: items})
		case http.MethodPost:
			var request dto.CreateAgentJobRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, "invalid json body", http.StatusBadRequest)
				return
			}
			cmd, err := toCreateAgentJobCommand(request)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			job, err := agentJobs.Create(r.Context(), cmd)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			writeJSON(w, http.StatusAccepted, types.Result{Code: types.ErrorCodeOK, Data: job})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func AgentJobLeaseNextHandler(agentJobs inport.AgentJobManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.AgentJobLeaseRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		timestamp, err := parseOptionalTimestamp(request.Timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		job, err := agentJobs.LeaseNext(r.Context(), command.AgentJobLeaseNextCommand{
			WorkerID:   request.WorkerID,
			JobType:    request.JobType,
			LeaseToken: request.LeaseToken,
			TTLSeconds: request.TTLSeconds,
			Timestamp:  timestamp,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: job})
	})
}

func AgentJobLeaseWorkHandler(agentJobs inport.AgentJobManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.AgentJobLeaseWorkRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		timestamp, err := parseOptionalTimestamp(request.Timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		job, err := agentJobs.LeaseWork(r.Context(), command.AgentJobLeaseWorkCommand{
			WorkKind:    request.WorkKind,
			WorkID:      request.WorkID,
			AggregateID: request.AggregateID,
			Subject:     request.Subject,
			WorkerID:    request.WorkerID,
			LeaseToken:  request.LeaseToken,
			TTLSeconds:  request.TTLSeconds,
			Timestamp:   timestamp,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: job})
	})
}

func AgentJobRecoverExpiredHandler(agentJobs inport.AgentJobManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.RecoverExpiredAgentJobLeasesRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil && !errors.Is(err, io.EOF) {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		timestamp, err := parseOptionalTimestamp(request.Timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		result, err := agentJobs.RecoverExpiredLeases(r.Context(), command.RecoverExpiredAgentJobLeasesCommand{
			Limit:     request.Limit,
			Timestamp: timestamp,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: result})
	})
}

func KnowledgeWorkerDiagnosticsHandler(diagnostics inport.KnowledgeWorkerDiagnosticsViewer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		item, err := diagnostics.Get(r.Context(), query.KnowledgeWorkerDiagnosticsFilter{
			Limit:             parsePositiveInt(r.URL.Query().Get("limit"), 50, 200),
			StaleAfterSeconds: parsePositiveInt(r.URL.Query().Get("stale_after_seconds"), 900, 86400),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: item})
	})
}

func AgentJobEventsHandler(jobEvents inport.AgentJobEventViewer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		items, err := jobEvents.List(r.Context(), query.AgentJobEventFilter{
			JobID:     r.URL.Query().Get("job_id"),
			JobType:   r.URL.Query().Get("type"),
			EventType: r.URL.Query().Get("event"),
			Limit:     parsePositiveInt(r.URL.Query().Get("limit"), 50, 200),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: items})
	})
}

func AgentJobMetricsHandler(metrics inport.AgentJobMetricsViewer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		item, err := metrics.Get(r.Context(), query.AgentJobMetricsFilter{
			JobLimit:   parsePositiveInt(r.URL.Query().Get("job_limit"), 200, 200),
			EventLimit: parsePositiveInt(r.URL.Query().Get("event_limit"), 200, 200),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: item})
	})
}

func InboxMetricsHandler(metrics inport.InboxMetricsViewer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		channelKind := r.URL.Query().Get("channel_kind")
		if channelKind == "" {
			channelKind = r.URL.Query().Get("platform")
		}
		item, err := metrics.Get(r.Context(), query.InboxMetricsFilter{
			Limit:            parsePositiveInt(r.URL.Query().Get("limit"), 200, 200),
			ChannelKind:      channelKind,
			AccountID:        r.URL.Query().Get("account_id"),
			ConversationID:   r.URL.Query().Get("conversation_id"),
			ConversationType: r.URL.Query().Get("conversation_type"),
			DecisionAction:   r.URL.Query().Get("decision_action"),
			ObserveOnly:      r.URL.Query().Get("observe_only"),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: item})
	})
}

func InboundDedupeCheckHandler(manager inport.InboundDedupeManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if manager == nil {
			http.Error(w, "inbound dedupe disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.CheckInboundDedupeRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		timestamp, err := parseOptionalTimestamp(request.Timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		item, err := manager.Check(r.Context(), command.CheckInboundDedupeCommand{
			Scope:      request.Scope,
			MessageKey: request.MessageKey,
			TTLSeconds: request.TTLSeconds,
			Timestamp:  timestamp,
			Metadata:   request.Metadata,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: item})
	})
}

func InboundDedupeRecordsHandler(manager inport.InboundDedupeManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if manager == nil {
			http.Error(w, "inbound dedupe disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		items, err := manager.List(r.Context(), query.InboundDedupeFilter{
			Limit: parsePositiveInt(r.URL.Query().Get("limit"), 100, 1000),
			Scope: r.URL.Query().Get("scope"),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: items})
	})
}

func InboundDedupeMetricsHandler(manager inport.InboundDedupeManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if manager == nil {
			http.Error(w, "inbound dedupe disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		item, err := manager.Metrics(r.Context(), query.InboundDedupeMetricsFilter{
			Limit: parsePositiveInt(r.URL.Query().Get("limit"), 100, 1000),
			Scope: r.URL.Query().Get("scope"),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: item})
	})
}

func OutboxDeliveryEventsHandler(outboxEvents inport.OutboxDeliveryEventViewer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		items, err := outboxEvents.List(r.Context(), query.OutboxDeliveryEventFilter{
			DeliveryID: r.URL.Query().Get("delivery_id"),
			Status:     r.URL.Query().Get("status"),
			EventType:  r.URL.Query().Get("event"),
			Limit:      parsePositiveInt(r.URL.Query().Get("limit"), 50, 200),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: items})
	})
}

func OutboxMetricsHandler(metrics inport.OutboxMetricsViewer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		item, err := metrics.Get(r.Context(), query.OutboxMetricsFilter{
			DeliveryLimit: parsePositiveInt(r.URL.Query().Get("delivery_limit"), 200, 200),
			EventLimit:    parsePositiveInt(r.URL.Query().Get("event_limit"), 200, 200),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: item})
	})
}

func QueueBackendHandler(queueBackend inport.QueueBackendViewer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		item, err := queueBackend.Get(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: item})
	})
}

func RuntimeOverviewHandler(viewer inport.RuntimeOverviewViewer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if viewer == nil {
			http.Error(w, "runtime overview disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		item, err := viewer.Get(r.Context(), query.RuntimeOverviewFilter{
			Limit:             parsePositiveInt(r.URL.Query().Get("limit"), 200, 200),
			EventLimit:        parsePositiveInt(r.URL.Query().Get("event_limit"), 50, 200),
			StaleAfterSeconds: parsePositiveInt(r.URL.Query().Get("stale_after_seconds"), 900, 86400),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: item})
	})
}

func RuntimeWorkerDiagnosticsHandler(viewer inport.RuntimeWorkerDiagnosticsViewer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if viewer == nil {
			http.Error(w, "runtime worker diagnostics disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		item, err := viewer.GetRuntimeWorkers(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: item})
	})
}

func AgentWorkerStatusReportHandler(manager inport.AgentWorkerStatusManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if manager == nil {
			http.Error(w, "agent worker status disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.AgentWorkerStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		timestamp, err := parseOptionalTimestamp(request.Timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		item, err := manager.ReportAgentWorkerStatus(r.Context(), command.ReportAgentWorkerStatusCommand{
			WorkerID:        request.WorkerID,
			InstanceID:      request.InstanceID,
			WorkerType:      request.WorkerType,
			Status:          request.Status,
			CurrentJobID:    request.CurrentJobID,
			LastJobID:       request.LastJobID,
			LastError:       request.LastError,
			ProcessedTotal:  request.ProcessedTotal,
			FailedTotal:     request.FailedTotal,
			Source:          request.Source,
			Metadata:        request.Metadata,
			Timestamp:       timestamp,
			LeaseTTLSeconds: request.LeaseTTLSeconds,
		})
		if err != nil {
			if errors.Is(err, appservice.ErrAgentWorkerLeaseConflict) {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusAccepted, types.Result{Code: types.ErrorCodeOK, Data: item})
	})
}

func AgentWorkerStatusesHandler(manager inport.AgentWorkerStatusManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if manager == nil {
			http.Error(w, "agent worker status disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		item, err := manager.ListAgentWorkerStatuses(r.Context(), query.AgentWorkerStatusFilter{
			StaleAfterSeconds: parsePositiveInt(r.URL.Query().Get("stale_after_seconds"), 0, 86400),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: item})
	})
}

func ProactiveDeliveriesHandler(proactiveState inport.ProactiveStateManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proactiveState == nil {
			http.Error(w, "proactive state disabled", http.StatusNotImplemented)
			return
		}
		switch r.Method {
		case http.MethodGet:
			items, err := proactiveState.ListDeliveries(r.Context(), query.ProactiveDeliveryFilter{
				Limit:       parsePositiveInt(r.URL.Query().Get("limit"), 50, 200),
				SessionKey:  r.URL.Query().Get("session_key"),
				DeliveryKey: r.URL.Query().Get("delivery_key"),
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: items})
		case http.MethodPost:
			var request dto.RecordProactiveDeliveryRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, "invalid json body", http.StatusBadRequest)
				return
			}
			timestamp, err := parseOptionalTimestamp(request.Timestamp)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			record, err := proactiveState.RecordDelivery(r.Context(), command.RecordProactiveDeliveryCommand{
				SessionKey:  request.SessionKey,
				DeliveryKey: request.DeliveryKey,
				Timestamp:   timestamp,
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			writeJSON(w, http.StatusAccepted, types.Result{Code: types.ErrorCodeOK, Data: record})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func ProactiveDeliveryDuplicateHandler(proactiveState inport.ProactiveStateManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proactiveState == nil {
			http.Error(w, "proactive state disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		timestamp, err := parseOptionalTimestamp(r.URL.Query().Get("timestamp"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		duplicate, err := proactiveState.IsDeliveryDuplicate(r.Context(), command.CheckProactiveDeliveryDuplicateCommand{
			SessionKey:  r.URL.Query().Get("session_key"),
			DeliveryKey: r.URL.Query().Get("delivery_key"),
			WindowHours: parsePositiveInt(r.URL.Query().Get("window_hours"), 24, 8760),
			Timestamp:   timestamp,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: duplicate})
	})
}

func ProactiveDeliveryCountHandler(proactiveState inport.ProactiveStateManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proactiveState == nil {
			http.Error(w, "proactive state disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		timestamp, err := parseOptionalTimestamp(r.URL.Query().Get("timestamp"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		count, err := proactiveState.CountDeliveries(r.Context(), command.CountProactiveDeliveriesCommand{
			SessionKey:  r.URL.Query().Get("session_key"),
			WindowHours: parsePositiveInt(r.URL.Query().Get("window_hours"), 24, 8760),
			Timestamp:   timestamp,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: count})
	})
}

func ProactiveSeenItemsHandler(proactiveState inport.ProactiveStateManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proactiveState == nil {
			http.Error(w, "proactive state disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.MarkProactiveItemsRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		timestamp, err := parseOptionalTimestamp(request.Timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		record, err := proactiveState.MarkItemsSeen(r.Context(), command.MarkProactiveItemsSeenCommand{
			Entries:   proactiveSourceItemEntriesFromDTO(request.Entries),
			Timestamp: timestamp,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusAccepted, types.Result{Code: types.ErrorCodeOK, Data: record})
	})
}

func ProactiveSeenItemsSeenHandler(proactiveState inport.ProactiveStateManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proactiveState == nil {
			http.Error(w, "proactive state disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		timestamp, err := parseOptionalTimestamp(r.URL.Query().Get("timestamp"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		seen, err := proactiveState.IsItemSeen(r.Context(), command.CheckProactiveItemSeenCommand{
			SourceKey: r.URL.Query().Get("source_key"),
			ItemID:    r.URL.Query().Get("item_id"),
			TTLHours:  parsePositiveInt(r.URL.Query().Get("ttl_hours"), 24, 8760),
			Timestamp: timestamp,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: seen})
	})
}

func ProactiveRejectionCooldownsHandler(proactiveState inport.ProactiveStateManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proactiveState == nil {
			http.Error(w, "proactive state disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.MarkProactiveRejectionCooldownRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		timestamp, err := parseOptionalTimestamp(request.Timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		record, err := proactiveState.MarkRejectionCooldown(r.Context(), command.MarkProactiveRejectionCooldownCommand{
			Entries:   proactiveSourceItemEntriesFromDTO(request.Entries),
			Hours:     request.Hours,
			Timestamp: timestamp,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusAccepted, types.Result{Code: types.ErrorCodeOK, Data: record})
	})
}

func ProactiveRejectionCooldownsCooledHandler(proactiveState inport.ProactiveStateManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proactiveState == nil {
			http.Error(w, "proactive state disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		timestamp, err := parseOptionalTimestamp(r.URL.Query().Get("timestamp"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		cooled, err := proactiveState.IsRejectionCooled(r.Context(), command.CheckProactiveRejectionCooldownCommand{
			SourceKey: r.URL.Query().Get("source_key"),
			ItemID:    r.URL.Query().Get("item_id"),
			TTLHours:  parseNonNegativeInt(r.URL.Query().Get("ttl_hours"), 0),
			Timestamp: timestamp,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: cooled})
	})
}

func proactiveSourceItemEntriesFromDTO(items []dto.ProactiveSourceItemEntry) []command.ProactiveSourceItemEntry {
	entries := make([]command.ProactiveSourceItemEntry, 0, len(items))
	for _, item := range items {
		entries = append(entries, command.ProactiveSourceItemEntry{
			SourceKey: item.SourceKey,
			ItemID:    item.ItemID,
		})
	}
	return entries
}

func ProactiveContextOnlyHandler(proactiveState inport.ProactiveStateManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proactiveState == nil {
			http.Error(w, "proactive state disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.RecordProactiveSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		timestamp, err := parseOptionalTimestamp(request.Timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mark, err := proactiveState.RecordContextOnly(r.Context(), command.RecordProactiveContextOnlyCommand{
			SessionKey: request.SessionKey,
			Timestamp:  timestamp,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusAccepted, types.Result{Code: types.ErrorCodeOK, Data: mark})
	})
}

func ProactiveContextOnlyLastHandler(proactiveState inport.ProactiveStateManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proactiveState == nil {
			http.Error(w, "proactive state disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		mark, err := proactiveState.LastContextOnly(r.Context(), r.URL.Query().Get("session_key"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: mark})
	})
}

func ProactiveContextOnlyCountHandler(proactiveState inport.ProactiveStateManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proactiveState == nil {
			http.Error(w, "proactive state disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		timestamp, err := parseOptionalTimestamp(r.URL.Query().Get("timestamp"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		count, err := proactiveState.CountContextOnly(r.Context(), command.CountProactiveContextOnlyCommand{
			SessionKey:  r.URL.Query().Get("session_key"),
			WindowHours: parsePositiveInt(r.URL.Query().Get("window_hours"), 24, 8760),
			Timestamp:   timestamp,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: count})
	})
}

func ProactiveDriftRunsHandler(proactiveState inport.ProactiveStateManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proactiveState == nil {
			http.Error(w, "proactive state disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.RecordProactiveSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		timestamp, err := parseOptionalTimestamp(request.Timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mark, err := proactiveState.RecordDriftRun(r.Context(), command.RecordProactiveDriftRunCommand{
			SessionKey: request.SessionKey,
			Timestamp:  timestamp,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusAccepted, types.Result{Code: types.ErrorCodeOK, Data: mark})
	})
}

func ProactiveDriftRunLastHandler(proactiveState inport.ProactiveStateManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proactiveState == nil {
			http.Error(w, "proactive state disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		mark, err := proactiveState.LastDriftRun(r.Context(), r.URL.Query().Get("session_key"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: mark})
	})
}

func ProactiveDriftFinishHandler(proactiveState inport.ProactiveStateManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proactiveState == nil {
			http.Error(w, "proactive state disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.RecordProactiveDriftFinishRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		timestamp, err := parseOptionalTimestamp(request.Timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		view, err := proactiveState.RecordDriftFinish(r.Context(), command.RecordProactiveDriftFinishCommand{
			SkillUsed:     request.SkillUsed,
			OneLine:       request.OneLine,
			Next:          request.Next,
			MessageResult: request.MessageResult,
			Note:          request.Note,
			Timestamp:     timestamp,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusAccepted, types.Result{Code: types.ErrorCodeOK, Data: view})
	})
}

func ProactiveDriftSummaryHandler(proactiveState inport.ProactiveStateManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proactiveState == nil {
			http.Error(w, "proactive state disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		view, err := proactiveState.DriftSummary(r.Context(), parseNonNegativeInt(r.URL.Query().Get("limit"), 10))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: view})
	})
}

func ProactiveDriftSkillStateHandler(proactiveState inport.ProactiveStateManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proactiveState == nil {
			http.Error(w, "proactive state disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		skillName := strings.TrimPrefix(r.URL.Path, "/v1/proactive/drift/skills/")
		if decoded, err := url.PathUnescape(skillName); err == nil {
			skillName = decoded
		}
		view, err := proactiveState.DriftSkillState(r.Context(), skillName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: view})
	})
}

func ProactiveTickLogsHandler(proactiveState inport.ProactiveStateManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proactiveState == nil {
			http.Error(w, "proactive state disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		startedFrom, err := parseOptionalTimestamp(r.URL.Query().Get("started_from"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		startedTo, err := parseOptionalTimestamp(r.URL.Query().Get("started_to"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		view, err := proactiveState.ListTickLogs(r.Context(), query.ProactiveTickLogFilter{
			Limit:          parsePositiveInt(r.URL.Query().Get("limit"), 50, 200),
			Offset:         parseNonNegativeInt(r.URL.Query().Get("offset"), 0),
			SessionKey:     r.URL.Query().Get("session_key"),
			TerminalAction: r.URL.Query().Get("terminal_action"),
			GateExit:       r.URL.Query().Get("gate_exit"),
			Flow:           r.URL.Query().Get("flow"),
			StartedFrom:    startedFrom,
			StartedTo:      startedTo,
			SortBy:         r.URL.Query().Get("sort_by"),
			SortOrder:      r.URL.Query().Get("sort_order"),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: view})
	})
}

func ProactiveTickLogStartHandler(proactiveState inport.ProactiveStateManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proactiveState == nil {
			http.Error(w, "proactive state disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.RecordProactiveTickLogStartRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		startedAt, err := parseOptionalTimestamp(request.StartedAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		view, err := proactiveState.RecordTickLogStart(r.Context(), command.RecordProactiveTickLogStartCommand{
			TickID:     request.TickID,
			SessionKey: request.SessionKey,
			StartedAt:  startedAt,
			GateExit:   request.GateExit,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusAccepted, types.Result{Code: types.ErrorCodeOK, Data: view})
	})
}

func ProactiveTickLogFinishHandler(proactiveState inport.ProactiveStateManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proactiveState == nil {
			http.Error(w, "proactive state disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.RecordProactiveTickLogFinishRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		startedAt, err := parseOptionalTimestamp(request.StartedAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		finishedAt, err := parseOptionalTimestamp(request.FinishedAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		view, err := proactiveState.RecordTickLogFinish(r.Context(), command.RecordProactiveTickLogFinishCommand{
			TickID:         request.TickID,
			SessionKey:     request.SessionKey,
			StartedAt:      startedAt,
			FinishedAt:     finishedAt,
			GateExit:       request.GateExit,
			TerminalAction: request.TerminalAction,
			SkipReason:     request.SkipReason,
			StepsTaken:     request.StepsTaken,
			AlertCount:     request.AlertCount,
			ContentCount:   request.ContentCount,
			ContextCount:   request.ContextCount,
			InterestingIDs: request.InterestingIDs,
			DiscardedIDs:   request.DiscardedIDs,
			CitedIDs:       request.CitedIDs,
			DriftEntered:   request.DriftEntered,
			FinalMessage:   request.FinalMessage,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusAccepted, types.Result{Code: types.ErrorCodeOK, Data: view})
	})
}

func ProactiveTickStepLogHandler(proactiveState inport.ProactiveStateManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proactiveState == nil {
			http.Error(w, "proactive state disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.RecordProactiveTickStepLogRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		view, err := proactiveState.RecordTickStepLog(r.Context(), command.RecordProactiveTickStepLogCommand{
			TickID:              request.TickID,
			StepIndex:           request.StepIndex,
			Phase:               request.Phase,
			ToolName:            request.ToolName,
			ToolCallID:          request.ToolCallID,
			ToolArgs:            request.ToolArgs,
			ToolResultText:      request.ToolResultText,
			TerminalActionAfter: request.TerminalActionAfter,
			SkipReasonAfter:     request.SkipReasonAfter,
			InterestingIDsAfter: request.InterestingIDsAfter,
			DiscardedIDsAfter:   request.DiscardedIDsAfter,
			CitedIDsAfter:       request.CitedIDsAfter,
			FinalMessageAfter:   request.FinalMessageAfter,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusAccepted, types.Result{Code: types.ErrorCodeOK, Data: view})
	})
}

func ProactiveTickLogDetailHandler(proactiveState inport.ProactiveStateManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proactiveState == nil {
			http.Error(w, "proactive state disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		tickID, steps := parseProactiveTickLogPath(r.URL.Path)
		if tickID == "" {
			http.Error(w, "tick_id required", http.StatusBadRequest)
			return
		}
		if steps {
			view, err := proactiveState.TickStepLogs(r.Context(), tickID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: view})
			return
		}
		view, err := proactiveState.TickLog(r.Context(), tickID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if !view.Found {
			http.Error(w, "tick log not found", http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: view})
	})
}

func parseProactiveTickLogPath(path string) (string, bool) {
	raw := strings.TrimPrefix(path, "/v1/proactive/tick-logs/")
	if raw == "" || raw == path {
		return "", false
	}
	steps := false
	if strings.HasSuffix(raw, "/steps") {
		steps = true
		raw = strings.TrimSuffix(raw, "/steps")
	}
	if decoded, err := url.PathUnescape(raw); err == nil {
		raw = decoded
	}
	return strings.TrimSpace(raw), steps
}

func ProactiveBGContextMainHandler(proactiveState inport.ProactiveStateManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proactiveState == nil {
			http.Error(w, "proactive state disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.RecordProactiveTimestampRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		timestamp, err := parseOptionalTimestamp(request.Timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mark, err := proactiveState.RecordBGContextMain(r.Context(), command.RecordProactiveBGContextMainCommand{
			Timestamp: timestamp,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusAccepted, types.Result{Code: types.ErrorCodeOK, Data: mark})
	})
}

func ProactiveBGContextMainLastHandler(proactiveState inport.ProactiveStateManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proactiveState == nil {
			http.Error(w, "proactive state disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		mark, err := proactiveState.LastBGContextMain(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: mark})
	})
}

func ProactiveAnyActionQuotaHandler(proactiveState inport.ProactiveStateManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proactiveState == nil {
			http.Error(w, "proactive state disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		timestamp, err := parseOptionalTimestamp(r.URL.Query().Get("timestamp"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		record, err := proactiveState.SnapshotAnyActionQuota(r.Context(), command.SnapshotProactiveAnyActionQuotaCommand{
			QuotaKey:  r.URL.Query().Get("quota_key"),
			ResetHour: parseNonNegativeInt(r.URL.Query().Get("reset_hour"), 12),
			Timezone:  r.URL.Query().Get("timezone"),
			Timestamp: timestamp,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: record})
	})
}

func ProactiveAnyActionRecordHandler(proactiveState inport.ProactiveStateManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proactiveState == nil {
			http.Error(w, "proactive state disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.ProactiveAnyActionQuotaRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		timestamp, err := parseOptionalTimestamp(request.Timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		record, err := proactiveState.RecordAnyAction(r.Context(), command.RecordProactiveAnyActionCommand{
			QuotaKey:  request.QuotaKey,
			ResetHour: request.ResetHour,
			Timezone:  request.Timezone,
			Timestamp: timestamp,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusAccepted, types.Result{Code: types.ErrorCodeOK, Data: record})
	})
}

func ProactiveCleanupHandler(proactiveState inport.ProactiveStateManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proactiveState == nil {
			http.Error(w, "proactive state disabled", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request dto.CleanupProactiveStateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		timestamp, err := parseOptionalTimestamp(request.Timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		result, err := proactiveState.Cleanup(r.Context(), command.CleanupProactiveStateCommand{
			SeenTTLHours:              request.SeenTTLHours,
			DeliveryTTLHours:          request.DeliveryTTLHours,
			ContextOnlyTTLHours:       request.ContextOnlyTTLHours,
			RejectionCooldownTTLHours: request.RejectionCooldownTTLHours,
			Timestamp:                 timestamp,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusAccepted, types.Result{Code: types.ErrorCodeOK, Data: result})
	})
}

func AgentJobStateHandler(agentJobs inport.AgentJobManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jobID, action := parseAgentJobPath(r.URL.Path)
		if jobID == "" {
			http.Error(w, "agent job id required", http.StatusBadRequest)
			return
		}
		if r.Method == http.MethodGet && action == "" {
			job, err := agentJobs.Get(r.Context(), jobID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: job})
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var state dto.AgentJobStateRequest
		var lease dto.AgentJobLeaseRequest
		if action == "lease" || action == "renew" {
			if err := json.NewDecoder(r.Body).Decode(&lease); err != nil {
				http.Error(w, "invalid json body", http.StatusBadRequest)
				return
			}
		} else {
			if err := json.NewDecoder(r.Body).Decode(&state); err != nil {
				http.Error(w, "invalid json body", http.StatusBadRequest)
				return
			}
		}

		var timestampText string
		if action == "lease" || action == "renew" {
			timestampText = lease.Timestamp
		} else {
			timestampText = state.Timestamp
		}
		timestamp, err := parseOptionalTimestamp(timestampText)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var job query.AgentJobView
		switch action {
		case "lease":
			job, err = agentJobs.Lease(r.Context(), command.AgentJobLeaseCommand{
				JobID:      jobID,
				WorkerID:   lease.WorkerID,
				LeaseToken: lease.LeaseToken,
				TTLSeconds: lease.TTLSeconds,
				Timestamp:  timestamp,
			})
		case "renew":
			job, err = agentJobs.RenewLease(r.Context(), command.RenewAgentJobLeaseCommand{
				JobID:      jobID,
				LeaseToken: lease.LeaseToken,
				TTLSeconds: lease.TTLSeconds,
				Timestamp:  timestamp,
			})
		case "running":
			job, err = agentJobs.MarkRunning(r.Context(), command.MarkAgentJobRunningCommand{
				JobID:      jobID,
				LeaseToken: state.LeaseToken,
				Timestamp:  timestamp,
			})
		case "succeeded":
			job, err = agentJobs.Complete(r.Context(), command.CompleteAgentJobCommand{
				JobID:      jobID,
				LeaseToken: state.LeaseToken,
				Result:     state.Result,
				Timestamp:  timestamp,
			})
		case "failed":
			job, err = agentJobs.Fail(r.Context(), command.FailAgentJobCommand{
				JobID:        jobID,
				LeaseToken:   state.LeaseToken,
				ErrorMessage: state.ErrorMessage,
				Timestamp:    timestamp,
			})
		case "retry":
			job, err = agentJobs.Retry(r.Context(), command.RetryAgentJobCommand{JobID: jobID, Timestamp: timestamp})
		case "cancel":
			job, err = agentJobs.Cancel(r.Context(), command.CancelAgentJobCommand{JobID: jobID, Timestamp: timestamp})
		default:
			http.Error(w, "unknown agent job action", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{Code: types.ErrorCodeOK, Data: job})
	})
}

func parseAgentJobPath(path string) (string, string) {
	rest := strings.TrimPrefix(path, "/v1/jobs/")
	rest = strings.Trim(rest, "/")
	if rest == "" {
		return "", ""
	}
	parts := strings.Split(rest, "/")
	jobID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	return jobID, action
}

func ImageJobsHandler(imageJobs inport.ImageJobManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request dto.CreateImageJobRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}

		cmd, err := toCreateImageJobCommand(request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		job, err := imageJobs.Create(r.Context(), cmd)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, http.StatusAccepted, types.Result{
			Code: types.ErrorCodeOK,
			Data: toImageJobResponse(job),
		})
	})
}

func ImageJobStateHandler(imageJobs inport.ImageJobManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jobID, action := parseImageJobPath(r.URL.Path)
		if jobID == "" {
			http.Error(w, "image job id required", http.StatusBadRequest)
			return
		}

		if r.Method == http.MethodGet && action == "" {
			job, err := imageJobs.Get(r.Context(), jobID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			writeJSON(w, http.StatusOK, types.Result{
				Code: types.ErrorCodeOK,
				Data: toImageJobResponse(job),
			})
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request dto.ImageJobStateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		timestamp, err := parseOptionalTimestamp(request.Timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var (
			job query.ImageJobView
		)
		switch action {
		case "running":
			job, err = imageJobs.MarkRunning(r.Context(), command.MarkImageJobRunningCommand{
				JobID:     jobID,
				Timestamp: timestamp,
			})
		case "succeeded":
			job, err = imageJobs.Complete(r.Context(), command.CompleteImageJobCommand{
				JobID:     jobID,
				Results:   toAttachmentCommands(request.Results),
				Timestamp: timestamp,
				Metadata:  request.Metadata,
			})
		case "failed":
			job, err = imageJobs.Fail(r.Context(), command.FailImageJobCommand{
				JobID:        jobID,
				ErrorMessage: request.ErrorMessage,
				Timestamp:    timestamp,
			})
		default:
			http.Error(w, "unknown image job action", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, types.Result{
			Code: types.ErrorCodeOK,
			Data: toImageJobResponse(job),
		})
	})
}

func toIngestCommand(request dto.IngestMessageRequest) (command.IngestMessageCommand, error) {
	timestamp := time.Now().UTC()
	if request.Timestamp != "" {
		parsed, err := time.Parse(time.RFC3339Nano, request.Timestamp)
		if err != nil {
			return command.IngestMessageCommand{}, err
		}
		timestamp = parsed
	}

	attachments := make([]command.AttachmentCommand, 0, len(request.Attachments))
	for _, item := range request.Attachments {
		attachments = append(attachments, command.AttachmentCommand{
			ID:        item.ID,
			Kind:      item.Kind,
			URL:       item.URL,
			MimeType:  item.MimeType,
			Name:      item.Name,
			SizeBytes: item.SizeBytes,
		})
	}

	return command.IngestMessageCommand{
		EventID: request.EventID,
		Channel: command.ChannelCommand{
			Kind:             request.Channel.RoutePlatform(),
			AccountID:        request.Channel.AccountID,
			ConversationID:   request.Channel.ConversationID,
			ConversationType: request.Channel.ConversationType,
		},
		Sender: command.SenderCommand{
			ID:          request.Sender.ID,
			DisplayName: request.Sender.DisplayName,
			Kind:        request.Sender.Kind,
		},
		Content:     request.Content,
		Attachments: attachments,
		Timestamp:   timestamp,
		Metadata:    request.Metadata,
	}, nil
}

func toSendCommand(request dto.SendMessageRequest) (command.SendMessageCommand, error) {
	timestamp := time.Now().UTC()
	if request.Timestamp != "" {
		parsed, err := time.Parse(time.RFC3339Nano, request.Timestamp)
		if err != nil {
			return command.SendMessageCommand{}, err
		}
		timestamp = parsed
	}

	attachments := make([]command.AttachmentCommand, 0, len(request.Attachments))
	for _, item := range request.Attachments {
		attachments = append(attachments, command.AttachmentCommand{
			ID:        item.ID,
			Kind:      item.Kind,
			URL:       item.URL,
			MimeType:  item.MimeType,
			Name:      item.Name,
			SizeBytes: item.SizeBytes,
		})
	}

	return command.SendMessageCommand{
		EventID: request.EventID,
		Channel: command.ChannelCommand{
			Kind:             request.Channel.RoutePlatform(),
			AccountID:        request.Channel.AccountID,
			ConversationID:   request.Channel.ConversationID,
			ConversationType: request.Channel.ConversationType,
		},
		Content:         request.Content,
		Attachments:     attachments,
		Timestamp:       timestamp,
		Metadata:        request.Metadata,
		WithBotProtocol: request.WithBotProtocol,
		ProtocolFromBot: request.ProtocolFromBot,
		ProtocolNonce:   request.ProtocolNonce,
		ProtocolNextHop: request.ProtocolNextHop,
	}, nil
}

func toCreateImageJobCommand(request dto.CreateImageJobRequest) (command.CreateImageJobCommand, error) {
	timestamp := time.Now().UTC()
	if request.Timestamp != "" {
		parsed, err := time.Parse(time.RFC3339Nano, request.Timestamp)
		if err != nil {
			return command.CreateImageJobCommand{}, err
		}
		timestamp = parsed
	}

	return command.CreateImageJobCommand{
		RequestID: request.RequestID,
		Requester: command.ChannelCommand{
			Kind:             request.Requester.RoutePlatform(),
			AccountID:        request.Requester.AccountID,
			ConversationID:   request.Requester.ConversationID,
			ConversationType: request.Requester.ConversationType,
		},
		RequesterID: request.RequesterID,
		Prompt:      request.Prompt,
		Provider:    request.Provider,
		Model:       request.Model,
		Size:        request.Size,
		Count:       request.Count,
		MaxAttempts: request.MaxAttempts,
		Timestamp:   timestamp,
		Metadata:    request.Metadata,
	}, nil
}

func toRegisterMediaAssetCommand(request dto.RegisterMediaAssetRequest) (command.RegisterMediaAssetCommand, error) {
	timestamp := time.Now().UTC()
	if request.Timestamp != "" {
		parsed, err := time.Parse(time.RFC3339Nano, request.Timestamp)
		if err != nil {
			return command.RegisterMediaAssetCommand{}, err
		}
		timestamp = parsed
	}

	return command.RegisterMediaAssetCommand{
		AssetID: request.AssetID,
		Channel: command.ChannelCommand{
			Kind:             request.Channel.RoutePlatform(),
			AccountID:        request.Channel.AccountID,
			ConversationID:   request.Channel.ConversationID,
			ConversationType: request.Channel.ConversationType,
		},
		SourceMessageID: request.SourceMessageID,
		SenderID:        request.SenderID,
		Kind:            request.Kind,
		URL:             request.URL,
		MimeType:        request.MimeType,
		Name:            request.Name,
		SizeBytes:       request.SizeBytes,
		ContentHash:     request.ContentHash,
		Retention:       request.Retention,
		Index:           request.Index,
		Timestamp:       timestamp,
		Metadata:        request.Metadata,
	}, nil
}

func toCreateAgentJobCommand(request dto.CreateAgentJobRequest) (command.CreateAgentJobCommand, error) {
	timestamp := time.Now().UTC()
	if request.Timestamp != "" {
		parsed, err := time.Parse(time.RFC3339Nano, request.Timestamp)
		if err != nil {
			return command.CreateAgentJobCommand{}, err
		}
		timestamp = parsed
	}

	return command.CreateAgentJobCommand{
		JobID:   request.JobID,
		JobType: request.JobType,
		AgentID: request.AgentID,
		Route: command.ChannelCommand{
			Kind:             request.Route.RoutePlatform(),
			AccountID:        request.Route.AccountID,
			ConversationID:   request.Route.ConversationID,
			ConversationType: request.Route.ConversationType,
		},
		SourceEventIDs: request.SourceEventIDs,
		SourceAssetIDs: request.SourceAssetIDs,
		Payload:        request.Payload,
		DedupeKey:      request.DedupeKey,
		MaxAttempts:    request.MaxAttempts,
		Timestamp:      timestamp,
		Metadata:       request.Metadata,
	}, nil
}

func toReplaceSchedulerJobsCommand(request dto.SchedulerJobSnapshotRequest) (command.ReplaceSchedulerJobsCommand, error) {
	jobs := make([]command.SchedulerJobCommand, 0, len(request.Jobs))
	for _, item := range request.Jobs {
		job, err := toSchedulerJobCommand(item)
		if err != nil {
			return command.ReplaceSchedulerJobsCommand{}, err
		}
		jobs = append(jobs, job)
	}
	return command.ReplaceSchedulerJobsCommand{
		Jobs:   jobs,
		Source: request.Source,
	}, nil
}

func toSchedulerJobCommand(item dto.SchedulerJobDTO) (command.SchedulerJobCommand, error) {
	fireAt, err := parseRequiredTimestamp(item.FireAt, "fire_at")
	if err != nil {
		return command.SchedulerJobCommand{}, err
	}
	createdAt, err := parseOptionalTimestamp(item.CreatedAt)
	if err != nil {
		return command.SchedulerJobCommand{}, err
	}
	enabled := true
	if item.Enabled != nil {
		enabled = *item.Enabled
	}
	return command.SchedulerJobCommand{
		ID:              item.ID,
		Trigger:         item.Trigger,
		Tier:            item.Tier,
		FireAt:          fireAt,
		Channel:         item.Channel,
		ChatID:          item.ChatID,
		IntervalSeconds: item.IntervalSeconds,
		CronExpr:        item.CronExpr,
		Message:         item.Message,
		Prompt:          item.Prompt,
		Name:            item.Name,
		Timezone:        item.Timezone,
		CreatedAt:       createdAt,
		RunCount:        item.RunCount,
		Enabled:         enabled,
	}, nil
}

func parseOptionalTimestamp(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, err
	}
	return parsed, nil
}

func parseRequiredTimestamp(value string, field string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, errors.New(field + " is required")
	}
	return time.Parse(time.RFC3339Nano, value)
}

func toAttachmentCommands(items []dto.AttachmentDTO) []command.AttachmentCommand {
	attachments := make([]command.AttachmentCommand, 0, len(items))
	for _, item := range items {
		attachments = append(attachments, command.AttachmentCommand{
			ID:        item.ID,
			Kind:      item.Kind,
			URL:       item.URL,
			MimeType:  item.MimeType,
			Name:      item.Name,
			SizeBytes: item.SizeBytes,
		})
	}
	return attachments
}

func parseImageJobPath(path string) (string, string) {
	rest := strings.TrimPrefix(path, "/v1/image-jobs/")
	rest = strings.Trim(rest, "/")
	if rest == "" {
		return "", ""
	}
	parts := strings.Split(rest, "/")
	jobID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	return jobID, action
}

func toImageJobResponse(job query.ImageJobView) dto.ImageJobResponse {
	results := make([]dto.AttachmentDTO, 0, len(job.Results))
	for _, item := range job.Results {
		results = append(results, dto.AttachmentDTO{
			ID:        item.ID,
			Kind:      item.Kind,
			URL:       item.URL,
			MimeType:  item.MimeType,
			Name:      item.Name,
			SizeBytes: item.SizeBytes,
		})
	}
	return dto.ImageJobResponse{
		JobID:     job.JobID,
		RequestID: job.RequestID,
		Requester: dto.ChannelDTO{
			Kind:             job.Requester.Kind,
			AccountID:        job.Requester.AccountID,
			ConversationID:   job.Requester.ConversationID,
			ConversationType: job.Requester.ConversationType,
		},
		RequesterID:  job.RequesterID,
		Prompt:       job.Prompt,
		Provider:     job.Provider,
		Model:        job.Model,
		Size:         job.Size,
		Count:        job.Count,
		Status:       job.Status,
		Attempts:     job.Attempts,
		MaxAttempts:  job.MaxAttempts,
		Results:      results,
		ErrorMessage: job.ErrorMessage,
		CreatedAt:    job.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:    job.UpdatedAt.Format(time.RFC3339Nano),
		Metadata:     job.Metadata,
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
