package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type AgentJobCapacityPlanner interface {
	PlanAgentJobCapacity(ctx context.Context, cmd command.PlanAgentJobCapacityCommand) (query.AgentJobCapacityPlanView, error)
}
