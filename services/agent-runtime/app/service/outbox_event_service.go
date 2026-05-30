package service

import (
	"context"
	"errors"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/assembler"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type OutboxDeliveryEventService struct {
	store outport.OutboxDeliveryEventStore
}

func NewOutboxDeliveryEventService(store outport.OutboxDeliveryEventStore) *OutboxDeliveryEventService {
	return &OutboxDeliveryEventService{store: store}
}

func (s *OutboxDeliveryEventService) List(ctx context.Context, filter query.OutboxDeliveryEventFilter) ([]query.OutboxDeliveryEventView, error) {
	if s == nil || s.store == nil {
		return nil, errors.New("outbox delivery event service requires store")
	}
	items, err := s.store.ListOutboxDeliveryEvents(ctx, filter)
	if err != nil {
		return nil, err
	}
	return assembler.ToOutboxDeliveryEventViews(items), nil
}
