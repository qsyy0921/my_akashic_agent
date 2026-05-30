package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type DeliveryDispatchPlanner interface {
	Plan(ctx context.Context, cmd command.PlanDeliveryDispatchCommand) (query.DeliveryDispatchPlanView, error)
}
