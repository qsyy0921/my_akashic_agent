package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type WorkQueueLeaseExecutor interface {
	ExecuteWorkQueueLease(ctx context.Context, cmd command.ExecuteWorkQueueLeaseCommand) (query.QueueExternalLeaseExecutionView, error)
}
