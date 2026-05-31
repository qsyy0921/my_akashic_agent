package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type AgentWorkerStatusManager interface {
	ReportAgentWorkerStatus(ctx context.Context, cmd command.ReportAgentWorkerStatusCommand) (query.AgentWorkerStatusView, error)
	ListAgentWorkerStatuses(ctx context.Context, filter query.AgentWorkerStatusFilter) (query.AgentWorkerStatusesView, error)
}
