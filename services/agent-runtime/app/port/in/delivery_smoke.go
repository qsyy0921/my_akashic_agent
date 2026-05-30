package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type DeliverySmokeReadinessChecker interface {
	CheckDeliverySmokeReadiness(ctx context.Context, cmd command.CheckDeliverySmokeReadinessCommand) (query.DeliverySmokeReadinessView, error)
}
