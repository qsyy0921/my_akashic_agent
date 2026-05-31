package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type SchedulerJobManager interface {
	ReplaceSchedulerJobs(ctx context.Context, cmd command.ReplaceSchedulerJobsCommand) (query.SchedulerJobSnapshotView, error)
	ListSchedulerJobs(ctx context.Context) ([]query.SchedulerJobView, error)
}
