package outport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type SchedulerJobRepository interface {
	ReplaceSchedulerJobs(ctx context.Context, jobs []model.SchedulerJob) error
	UpsertSchedulerJob(ctx context.Context, job model.SchedulerJob) (created bool, err error)
	DeleteSchedulerJob(ctx context.Context, jobID string) (found bool, err error)
	ListSchedulerJobs(ctx context.Context) ([]model.SchedulerJob, error)
}
