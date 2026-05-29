package httptrigger

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/api/dto"
	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/app/command"
	inport "github.com/kachofugetsu09/akashic-agent/services/message-gateway/app/port/in"
	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/types"
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
) {
	mux.Handle("/healthz", HealthHandler())
	mux.Handle("/v1/inbound", IngestHandler(ingestor))
	mux.Handle("/v1/shadow/inbound", ShadowIngestHandler(shadowIngestor))
	mux.Handle("/v1/shadow/observed", ShadowObservedHandler(shadowViewer))
	mux.Handle("/v1/outbound", SendHandler(sender))
	mux.Handle("/v1/image-jobs", ImageJobsHandler(imageJobs))
	mux.Handle("/v1/image-jobs/", ImageJobStateHandler(imageJobs))
	mux.Handle("/v1/outbox", OutboxListHandler(outbox))
	mux.Handle("/v1/outbox/", OutboxStateHandler(outbox))
	mux.Handle("/v1/media-assets", MediaAssetsHandler(mediaAssets))
	mux.Handle("/v1/media-assets/", MediaAssetStateHandler(mediaAssets))
	mux.Handle("/v1/jobs", AgentJobsHandler(agentJobs))
	mux.Handle("/v1/jobs/lease-next", AgentJobLeaseNextHandler(agentJobs))
	mux.Handle("/v1/jobs/", AgentJobStateHandler(agentJobs))
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
			limit := parsePositiveInt(r.URL.Query().Get("limit"), 50, 200)
			items, err := mediaAssets.List(r.Context(), limit)
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
			http.Error(w, "media asset content route is not enabled", http.StatusNotImplemented)
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
