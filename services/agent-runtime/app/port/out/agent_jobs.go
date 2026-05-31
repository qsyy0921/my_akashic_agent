package outport

import (
	"context"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type AgentJobRepository interface {
	SaveAgentJob(ctx context.Context, job model.AgentJob) error
	FindAgentJob(ctx context.Context, jobID string) (model.AgentJob, bool, error)
	FindActiveAgentJobByDedupeKey(ctx context.Context, jobType string, dedupeKey string, now time.Time) (model.AgentJob, bool, error)
	ListAgentJobs(ctx context.Context, filter query.AgentJobFilter) ([]model.AgentJob, error)
	FindLeaseableAgentJob(ctx context.Context, jobType string, now time.Time) (model.AgentJob, bool, error)
	ListExpiredAgentJobLeases(ctx context.Context, now time.Time, limit int) ([]model.AgentJob, error)
}

type AgentJobEventSink interface {
	AppendAgentJobEvent(ctx context.Context, event model.AgentJobEvent) error
}

type AgentJobEventStore interface {
	AgentJobEventSink
	ListAgentJobEvents(ctx context.Context, filter query.AgentJobEventFilter) ([]model.AgentJobEvent, error)
}
