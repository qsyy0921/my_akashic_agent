package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type SchedulerJobManager interface {
	ReplaceSchedulerJobs(ctx context.Context, cmd command.ReplaceSchedulerJobsCommand) (query.SchedulerJobSnapshotView, error)
	UpsertSchedulerJob(ctx context.Context, cmd command.UpsertSchedulerJobCommand) (query.SchedulerJobMutationView, error)
	DeleteSchedulerJob(ctx context.Context, cmd command.DeleteSchedulerJobCommand) (query.SchedulerJobMutationView, error)
	ListSchedulerJobs(ctx context.Context) ([]query.SchedulerJobView, error)
	AcquireSchedulerExecutionLease(ctx context.Context, cmd command.AcquireSchedulerExecutionLeaseCommand) (query.SchedulerExecutionLeaseView, error)
	RenewSchedulerExecutionLease(ctx context.Context, cmd command.RenewSchedulerExecutionLeaseCommand) (query.SchedulerExecutionLeaseView, error)
	ReleaseSchedulerExecutionLease(ctx context.Context, cmd command.ReleaseSchedulerExecutionLeaseCommand) (query.SchedulerExecutionLeaseView, error)
	ListSchedulerExecutionLeases(ctx context.Context) (query.SchedulerExecutionLeasesView, error)
}
