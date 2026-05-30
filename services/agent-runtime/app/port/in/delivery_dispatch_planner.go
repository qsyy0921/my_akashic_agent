package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type DeliveryDispatchPlanner interface {
	Plan(ctx context.Context, cmd command.PlanDeliveryDispatchCommand) (query.DeliveryDispatchPlanView, error)
	Readiness(ctx context.Context, cmd command.CheckDeliveryDispatchReadinessCommand) (query.DeliveryDispatchReadinessView, error)
	Dispatch(ctx context.Context, cmd command.DispatchDeliveryCommand) (query.DeliveryDispatchResultView, error)
}
