package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type ProactiveStateManager interface {
	RecordDelivery(ctx context.Context, cmd command.RecordProactiveDeliveryCommand) (query.ProactiveDeliveryView, error)
	IsDeliveryDuplicate(ctx context.Context, cmd command.CheckProactiveDeliveryDuplicateCommand) (query.ProactiveDuplicateView, error)
	CountDeliveries(ctx context.Context, cmd command.CountProactiveDeliveriesCommand) (query.ProactiveCountView, error)
	ListDeliveries(ctx context.Context, filter query.ProactiveDeliveryFilter) ([]query.ProactiveDeliveryView, error)

	RecordContextOnly(ctx context.Context, cmd command.RecordProactiveContextOnlyCommand) (query.ProactiveTimestampView, error)
	LastContextOnly(ctx context.Context, sessionKey string) (query.ProactiveTimestampView, error)
	CountContextOnly(ctx context.Context, cmd command.CountProactiveContextOnlyCommand) (query.ProactiveCountView, error)

	RecordDriftRun(ctx context.Context, cmd command.RecordProactiveDriftRunCommand) (query.ProactiveTimestampView, error)
	LastDriftRun(ctx context.Context, sessionKey string) (query.ProactiveTimestampView, error)
}
