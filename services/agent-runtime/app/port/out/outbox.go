package outport

import (
	"context"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type OutboxRepository interface {
	SaveOutboxDelivery(ctx context.Context, delivery model.OutboxDelivery) error
	FindOutboxDelivery(ctx context.Context, eventID string) (model.OutboxDelivery, bool, error)
	ListOutboxDeliveries(ctx context.Context, limit int) ([]model.OutboxDelivery, error)
	FindLeaseableOutboxDelivery(ctx context.Context, filter OutboxLeaseFilter) (model.OutboxDelivery, bool, error)
}

type OutboxLeaseFilter struct {
	Now                time.Time
	BlockedAccountKeys map[string]struct{}
}

func (f OutboxLeaseFilter) Allows(delivery model.OutboxDelivery) bool {
	if len(f.BlockedAccountKeys) == 0 {
		return true
	}
	_, blocked := f.BlockedAccountKeys[delivery.Message.Channel.AccountKey()]
	return !blocked
}

type OutboxQueue interface {
	EnqueueOutboxDelivery(ctx context.Context, delivery model.OutboxDelivery) error
}

type OutboxDeliveryEventSink interface {
	AppendOutboxDeliveryEvent(ctx context.Context, event model.OutboxDeliveryEvent) error
}

type OutboxDeliveryEventStore interface {
	OutboxDeliveryEventSink
	ListOutboxDeliveryEvents(ctx context.Context, filter query.OutboxDeliveryEventFilter) ([]model.OutboxDeliveryEvent, error)
}
