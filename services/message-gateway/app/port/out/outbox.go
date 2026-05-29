package outport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/domain/model"
)

type OutboxRepository interface {
	SaveOutboxDelivery(ctx context.Context, delivery model.OutboxDelivery) error
	FindOutboxDelivery(ctx context.Context, eventID string) (model.OutboxDelivery, bool, error)
	ListOutboxDeliveries(ctx context.Context, limit int) ([]model.OutboxDelivery, error)
}

type OutboxQueue interface {
	EnqueueOutboxDelivery(ctx context.Context, delivery model.OutboxDelivery) error
}
