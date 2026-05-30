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
	mux.Handle("/v1/jobs/", AgentJobStateHandler(agentJobs))
	mux.Handle("/v1/send-ledger/records", SendLedgerRecordsHandler(sendLedger))
	mux.Handle("/v1/send-ledger/recent", SendLedgerRecentHandler(sendLedger))
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

func RegisterAgentJobEventRoutes(
	mux *http.ServeMux,
	jobEvents inport.AgentJobEventViewer,
) {
	mux.Handle("/v1/job-events", AgentJobEventsHandler(jobEvents))
}

func RegisterOutboxEventRoutes(
	mux *http.ServeMux,
	outboxEvents inport.OutboxDeliveryEventViewer,
) {
	mux.Handle("/v1/outbox-events", OutboxDeliveryEventsHandler(outboxEvents))
}

func RegisterDeliveryDispatchRoutes(
	mux *http.ServeMux,
	planner inport.DeliveryDispatchPlanner,
) {
	mux.Handle("/v1/delivery-dispatch/plan", DeliveryDispatchPlanHandler(planner))
	mux.Handle("/v1/delivery-dispatch/send", DeliveryDispatchSendHandler(planner))
}

func RegisterProactiveStateRoutes(
	mux *http.ServeMux,
	proactiveState inport.ProactiveStateManager,
) {
	mux.Handle("/v1/proactive/deliveries", ProactiveDeliveriesHandler(proactiveState))
	mux.Handle("/v1/proactive/deliveries/duplicate", ProactiveDeliveryDuplicateHandler(proactiveState))
	mux.Handle("/v1/proactive/deliveries/count", ProactiveDeliveryCountHandler(proactiveState))
	mux.Handle("/v1/proactive/context-only", ProactiveContextOnlyHandler(proactiveState))
	mux.Handle("/v1/proactive/context-only/last", ProactiveContextOnlyLastHandler(proactiveState))
	mux.Handle("/v1/proactive/context-only/count", ProactiveContextOnlyCountHandler(proactiveState))
	mux.Handle("/v1/proactive/drift-runs", ProactiveDriftRunsHandler(proactiveState))
	mux.Handle("/v1/proactive/drift-runs/last", ProactiveDriftRunLastHandler(proactiveState))
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
		if action == "lease" {
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
		if action == "lease" {
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
				TTLSeconds: lease.TTLSeconds,
				Timestamp:  timestamp,
			})
		case "running":
			job, err = agentJobs.MarkRunning(r.Context(), command.MarkAgentJobRunningCommand{JobID: jobID, Timestamp: timestamp})
		case "succeeded":
			job, err = agentJobs.Complete(r.Context(), command.CompleteAgentJobCommand{JobID: jobID, Result: state.Result, Timestamp: timestamp})
		case "failed":
			job, err = agentJobs.Fail(r.Context(), command.FailAgentJobCommand{JobID: jobID, ErrorMessage: state.ErrorMessage, Timestamp: timestamp})
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
		MaxAttempts:    request.MaxAttempts,
		Timestamp:      timestamp,
		Metadata:       request.Metadata,
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
