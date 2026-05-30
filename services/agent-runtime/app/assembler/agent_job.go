package assembler

import (
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func ToAgentJobView(job model.AgentJob) query.AgentJobView {
	return query.AgentJobView{
		JobID:   job.JobID,
		JobType: string(job.JobType),
		AgentID: job.AgentID,
		Route: query.AgentJobRouteView{
			Kind:             string(job.Route.Kind),
			AccountID:        job.Route.AccountID,
			ConversationID:   job.Route.ConversationID,
			ConversationType: string(job.Route.ConversationType),
		},
		SourceEventIDs: append([]string(nil), job.SourceEventIDs...),
		SourceAssetIDs: append([]string(nil), job.SourceAssetIDs...),
		Payload:        job.Payload,
		Status:         string(job.Status),
		Attempts:       job.Attempts,
		MaxAttempts:    job.MaxAttempts,
		LeaseOwner:     job.LeaseOwner,
		LeaseToken:     job.LeaseToken,
		LeaseExpiresAt: formatAgentJobTime(job.LeaseExpiresAt),
		Result:         job.Result,
		ErrorMessage:   job.ErrorMessage,
		CreatedAt:      formatAgentJobTime(job.CreatedAt),
		UpdatedAt:      formatAgentJobTime(job.UpdatedAt),
		Metadata:       job.Metadata,
	}
}

func ToAgentJobViews(items []model.AgentJob) []query.AgentJobView {
	views := make([]query.AgentJobView, 0, len(items))
	for _, item := range items {
		views = append(views, ToAgentJobView(item))
	}
	return views
}

func formatAgentJobTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}
