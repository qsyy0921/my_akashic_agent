package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type KnowledgeJobPlannerPreviewer interface {
	PreviewKnowledgeJobs(ctx context.Context, cmd command.PlanKnowledgeJobsCommand) (query.KnowledgeJobPlannerPreviewView, error)
}
