package assembler

import (
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func ToAgentJobEventView(event model.AgentJobEvent) query.AgentJobEventView {
	return query.AgentJobEventView{
		EventID:        event.EventID,
		JobID:          event.JobID,
		JobType:        string(event.JobType),
		EventType:      string(event.EventType),
		Status:         string(event.Status),
		Attempt:        event.Attempt,
		MaxAttempts:    event.MaxAttempts,
		LeaseOwner:     event.LeaseOwner,
		LeaseExpiresAt: formatAgentJobTime(event.LeaseExpiresAt),
		OccurredAt:     formatAgentJobTime(event.OccurredAt),
		Metadata:       event.Metadata,
	}
}

func ToAgentJobEventViews(items []model.AgentJobEvent) []query.AgentJobEventView {
	views := make([]query.AgentJobEventView, 0, len(items))
	for _, item := range items {
		views = append(views, ToAgentJobEventView(item))
	}
	return views
}
