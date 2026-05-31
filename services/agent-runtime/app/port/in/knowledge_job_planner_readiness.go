package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type KnowledgeJobPlannerReadinessChecker interface {
	CheckKnowledgeJobPlannerReadiness(ctx context.Context, cmd command.CheckKnowledgeJobPlannerReadinessCommand) (query.KnowledgeJobPlannerReadinessView, error)
}
