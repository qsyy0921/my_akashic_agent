package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type ObserveTargetManager interface {
	SyncObserveTargets(ctx context.Context, cmd command.SyncObserveTargetsCommand) (query.ObserveTargetsView, error)
	ListObserveTargets(ctx context.Context) (query.ObserveTargetsView, error)
}
