package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type OutboundCutoverReadinessChecker interface {
	CheckOutboundCutoverReadiness(ctx context.Context, cmd command.CheckOutboundCutoverReadinessCommand) (query.OutboundCutoverReadinessView, error)
}

type OutboundCutoverPlanner interface {
	PlanOutboundCutover(ctx context.Context, cmd command.PlanOutboundCutoverCommand) (query.OutboundCutoverPlanView, error)
}
