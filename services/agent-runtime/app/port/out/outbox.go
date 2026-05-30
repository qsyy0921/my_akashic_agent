package outport

import (
	"context"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type OutboxRepository interface {
	SaveOutboxDelivery(ctx context.Context, delivery model.OutboxDelivery) error
	FindOutboxDelivery(ctx context.Context, eventID string) (model.OutboxDelivery, bool, error)
	ListOutboxDeliveries(ctx context.Context, limit int) ([]model.OutboxDelivery, error)
	FindLeaseableOutboxDelivery(ctx context.Context, now time.Time) (model.OutboxDelivery, bool, error)
}

type OutboxQueue interface {
	EnqueueOutboxDelivery(ctx context.Context, delivery model.OutboxDelivery) error
}
