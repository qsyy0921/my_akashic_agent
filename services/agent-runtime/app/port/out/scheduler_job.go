package outport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type SchedulerJobRepository interface {
	ReplaceSchedulerJobs(ctx context.Context, jobs []model.SchedulerJob) error
	ListSchedulerJobs(ctx context.Context) ([]model.SchedulerJob, error)
}
