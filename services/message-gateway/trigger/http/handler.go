package httptrigger

import (
	"encoding/json"
	"net/http"
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
	sender inport.MessageSender,
	imageJobs inport.ImageJobManager,
) {
	mux.Handle("/healthz", HealthHandler())
	mux.Handle("/v1/inbound", IngestHandler(ingestor))
	mux.Handle("/v1/shadow/inbound", ShadowIngestHandler(shadowIngestor))
	mux.Handle("/v1/outbound", SendHandler(sender))
	mux.Handle("/v1/image-jobs", ImageJobsHandler(imageJobs))
	mux.Handle("/v1/image-jobs/", ImageJobStateHandler(imageJobs))
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
