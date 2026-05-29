package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type AgentJobManager interface {
	Create(ctx context.Context, cmd command.CreateAgentJobCommand) (query.AgentJobView, error)
	List(ctx context.Context, filter query.AgentJobFilter) ([]query.AgentJobView, error)
	Get(ctx context.Context, jobID string) (query.AgentJobView, error)
	Lease(ctx context.Context, cmd command.AgentJobLeaseCommand) (query.AgentJobView, error)
	LeaseNext(ctx context.Context, cmd command.AgentJobLeaseNextCommand) (query.AgentJobView, error)
	MarkRunning(ctx context.Context, cmd command.MarkAgentJobRunningCommand) (query.AgentJobView, error)
	Complete(ctx context.Context, cmd command.CompleteAgentJobCommand) (query.AgentJobView, error)
	Fail(ctx context.Context, cmd command.FailAgentJobCommand) (query.AgentJobView, error)
	Retry(ctx context.Context, cmd command.RetryAgentJobCommand) (query.AgentJobView, error)
	Cancel(ctx context.Context, cmd command.CancelAgentJobCommand) (query.AgentJobView, error)
}

