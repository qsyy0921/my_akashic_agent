package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type OutboxManager interface {
	List(ctx context.Context, limit int) ([]query.OutboxDeliveryView, error)
	Get(ctx context.Context, eventID string) (query.OutboxDeliveryView, error)
	MarkDispatching(ctx context.Context, cmd command.MarkOutboxDispatchingCommand) (query.OutboxDeliveryView, error)
	MarkSucceeded(ctx context.Context, cmd command.MarkOutboxSucceededCommand) (query.OutboxDeliveryView, error)
	MarkFailed(ctx context.Context, cmd command.MarkOutboxFailedCommand) (query.OutboxDeliveryView, error)
	Retry(ctx context.Context, cmd command.RetryOutboxCommand) (query.OutboxDeliveryView, error)
}

