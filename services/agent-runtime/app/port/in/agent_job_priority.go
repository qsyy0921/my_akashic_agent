package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type AgentJobPriorityPlanner interface {
	PlanAgentJobPriority(ctx context.Context, cmd command.PlanAgentJobPriorityCommand) (query.AgentJobPriorityPlanView, error)
}
