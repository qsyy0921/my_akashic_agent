package outport

import (
	"context"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/domain/model"
)

type AgentJobRepository interface {
	SaveAgentJob(ctx context.Context, job model.AgentJob) error
	FindAgentJob(ctx context.Context, jobID string) (model.AgentJob, bool, error)
	ListAgentJobs(ctx context.Context, filter query.AgentJobFilter) ([]model.AgentJob, error)
	FindLeaseableAgentJob(ctx context.Context, jobType string, now time.Time) (model.AgentJob, bool, error)
}
