package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type KnowledgeJobPlannerCutoverPlanner interface {
	PlanKnowledgeJobPlannerCutover(ctx context.Context, cmd command.PlanKnowledgeJobPlannerCutoverCommand) (query.KnowledgeJobPlannerCutoverPlanView, error)
}
