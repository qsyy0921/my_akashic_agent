package assembler

import (
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func ToImageJobView(job model.ImageJob) query.ImageJobView {
	results := make([]query.AttachmentView, 0, len(job.Results))
	for _, item := range job.Results {
		results = append(results, query.AttachmentView{
			ID:        item.ID,
			Kind:      string(item.Kind),
			URL:       item.URL,
			MimeType:  item.MimeType,
			Name:      item.Name,
			SizeBytes: item.SizeBytes,
		})
	}
	return query.ImageJobView{
		JobID:     job.JobID,
		RequestID: job.RequestID,
		Requester: query.ChannelView{
			Kind:             string(job.Requester.Kind),
			AccountID:        job.Requester.AccountID,
			ConversationID:   job.Requester.ConversationID,
			ConversationType: string(job.Requester.ConversationType),
		},
		RequesterID:  job.RequesterID,
		Prompt:       job.Prompt,
		Provider:     job.Options.Provider,
		Model:        job.Options.Model,
		Size:         job.Options.Size,
		Count:        job.Options.Count,
		Status:       string(job.Status),
		Attempts:     job.Attempts,
		MaxAttempts:  job.MaxAttempts,
		Results:      results,
		ErrorMessage: job.ErrorMessage,
		CreatedAt:    job.CreatedAt,
		UpdatedAt:    job.UpdatedAt,
		Metadata:     job.Metadata,
	}
}

