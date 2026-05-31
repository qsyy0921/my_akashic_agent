package assembler

import (
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func ToAgentWorkerStatusView(item model.AgentWorkerStatus, stale bool) query.AgentWorkerStatusView {
	return query.AgentWorkerStatusView{
		WorkerID:       item.WorkerID,
		InstanceID:     item.InstanceID,
		WorkerType:     item.WorkerType,
		Status:         string(item.Status),
		CurrentJobID:   item.CurrentJobID,
		LastJobID:      item.LastJobID,
		LastError:      item.LastError,
		ProcessedTotal: item.ProcessedTotal,
		FailedTotal:    item.FailedTotal,
		Source:         item.Source,
		Metadata:       item.Metadata,
		UpdatedAt:      item.UpdatedAt.UTC().Format(time.RFC3339Nano),
		LeaseUntil:     formatAgentWorkerStatusTime(item.LeaseUntil),
		LeaseActive:    item.LeaseActive(time.Now().UTC()),
		Stale:          stale,
	}
}

func ToAgentWorkerStatusViews(items []model.AgentWorkerStatus, staleByWorkerID map[string]bool) []query.AgentWorkerStatusView {
	views := make([]query.AgentWorkerStatusView, 0, len(items))
	for _, item := range items {
		views = append(views, ToAgentWorkerStatusView(item, staleByWorkerID[item.WorkerID]))
	}
	return views
}

func formatAgentWorkerStatusTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
