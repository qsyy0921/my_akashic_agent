package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/assembler"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type OutboxService struct {
	repository outport.OutboxRepository
	queue      outport.OutboxQueue
	events     outport.OutboxDeliveryEventSink
}

func NewOutboxService(repository outport.OutboxRepository, queue outport.OutboxQueue) *OutboxService {
	return &OutboxService{repository: repository, queue: queue}
}

func NewOutboxServiceWithEvents(
	repository outport.OutboxRepository,
	queue outport.OutboxQueue,
	events outport.OutboxDeliveryEventSink,
) *OutboxService {
	return &OutboxService{repository: repository, queue: queue, events: events}
}

func (s *OutboxService) List(ctx context.Context, limit int) ([]query.OutboxDeliveryView, error) {
	if s == nil || s.repository == nil {
		return nil, errors.New("outbox service requires repository")
	}
	deliveries, err := s.repository.ListOutboxDeliveries(ctx, limit)
	if err != nil {
		return nil, err
	}
	return assembler.ToOutboxDeliveryViews(deliveries), nil
}

func (s *OutboxService) Get(ctx context.Context, eventID string) (query.OutboxDeliveryView, error) {
	delivery, err := s.getModel(ctx, eventID)
	if err != nil {
		return query.OutboxDeliveryView{}, err
	}
	return assembler.ToOutboxDeliveryView(delivery), nil
}

func (s *OutboxService) LeaseNext(ctx context.Context, cmd command.LeaseNextOutboxCommand) (query.OutboxDeliveryView, error) {
	if s == nil || s.repository == nil {
		return query.OutboxDeliveryView{}, errors.New("outbox service requires repository")
	}
	if cmd.Timestamp.IsZero() {
		cmd.Timestamp = time.Now().UTC()
	}
	ttl := time.Duration(cmd.TTLSeconds) * time.Second
	delivery, ok, err := s.repository.FindLeaseableOutboxDelivery(ctx, cmd.Timestamp)
	if err != nil {
		return query.OutboxDeliveryView{}, err
	}
	if !ok {
		return query.OutboxDeliveryView{}, errors.New("no leaseable outbox delivery")
	}
	if err := delivery.Lease(cmd.WorkerID, ttl, cmd.Timestamp); err != nil {
		return query.OutboxDeliveryView{}, err
	}
	if err := s.repository.SaveOutboxDelivery(ctx, delivery); err != nil {
		return query.OutboxDeliveryView{}, err
	}
	if err := s.recordEvent(ctx, delivery, model.OutboxDeliveryEventLeased, cmd.Timestamp); err != nil {
		return query.OutboxDeliveryView{}, err
	}
	return assembler.ToOutboxDeliveryView(delivery), nil
}

func (s *OutboxService) Lease(ctx context.Context, cmd command.LeaseOutboxDeliveryCommand) (query.OutboxDeliveryView, error) {
	if s == nil || s.repository == nil {
		return query.OutboxDeliveryView{}, errors.New("outbox service requires repository")
	}
	if cmd.Timestamp.IsZero() {
		cmd.Timestamp = time.Now().UTC()
	}
	ttl := time.Duration(cmd.TTLSeconds) * time.Second
	delivery, err := s.getModel(ctx, cmd.EventID)
	if err != nil {
		return query.OutboxDeliveryView{}, err
	}
	if err := delivery.Lease(cmd.WorkerID, ttl, cmd.Timestamp); err != nil {
		return query.OutboxDeliveryView{}, err
	}
	if err := s.repository.SaveOutboxDelivery(ctx, delivery); err != nil {
		return query.OutboxDeliveryView{}, err
	}
	if err := s.recordEvent(ctx, delivery, model.OutboxDeliveryEventLeased, cmd.Timestamp); err != nil {
		return query.OutboxDeliveryView{}, err
	}
	return assembler.ToOutboxDeliveryView(delivery), nil
}

func (s *OutboxService) MarkDispatching(ctx context.Context, cmd command.MarkOutboxDispatchingCommand) (query.OutboxDeliveryView, error) {
	return s.update(ctx, cmd.EventID, model.OutboxDeliveryEventDispatching, cmd.Timestamp, func(delivery *model.OutboxDelivery, now time.Time) error {
		return delivery.MarkDispatching(now)
	})
}

func (s *OutboxService) MarkSucceeded(ctx context.Context, cmd command.MarkOutboxSucceededCommand) (query.OutboxDeliveryView, error) {
	return s.update(ctx, cmd.EventID, model.OutboxDeliveryEventSucceeded, cmd.Timestamp, func(delivery *model.OutboxDelivery, now time.Time) error {
		return delivery.MarkSucceeded(now)
	})
}

func (s *OutboxService) MarkFailed(ctx context.Context, cmd command.MarkOutboxFailedCommand) (query.OutboxDeliveryView, error) {
	return s.update(ctx, cmd.EventID, model.OutboxDeliveryEventFailed, cmd.Timestamp, func(delivery *model.OutboxDelivery, now time.Time) error {
		return delivery.MarkFailedWithKind(
			model.NormalizeDeliveryErrorKind(cmd.ErrorKind),
			cmd.ErrorMessage,
			now,
		)
	})
}

func (s *OutboxService) Retry(ctx context.Context, cmd command.RetryOutboxCommand) (query.OutboxDeliveryView, error) {
	view, err := s.update(ctx, cmd.EventID, model.OutboxDeliveryEventRetry, cmd.Timestamp, func(delivery *model.OutboxDelivery, now time.Time) error {
		return delivery.Retry(now)
	})
	if err != nil {
		return query.OutboxDeliveryView{}, err
	}
	if s.queue != nil {
		delivery, getErr := s.getModel(ctx, cmd.EventID)
		if getErr != nil {
			return query.OutboxDeliveryView{}, getErr
		}
		if enqueueErr := s.queue.EnqueueOutboxDelivery(ctx, delivery); enqueueErr != nil {
			return query.OutboxDeliveryView{}, enqueueErr
		}
	}
	return view, nil
}

func (s *OutboxService) update(
	ctx context.Context,
	eventID string,
	eventType model.OutboxDeliveryEventType,
	timestamp time.Time,
	mutate func(delivery *model.OutboxDelivery, now time.Time) error,
) (query.OutboxDeliveryView, error) {
	if s == nil || s.repository == nil {
		return query.OutboxDeliveryView{}, errors.New("outbox service requires repository")
	}
	delivery, err := s.getModel(ctx, eventID)
	if err != nil {
		return query.OutboxDeliveryView{}, err
	}
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	if err := mutate(&delivery, timestamp); err != nil {
		return query.OutboxDeliveryView{}, err
	}
	if err := s.repository.SaveOutboxDelivery(ctx, delivery); err != nil {
		return query.OutboxDeliveryView{}, err
	}
	if err := s.recordEvent(ctx, delivery, eventType, timestamp); err != nil {
		return query.OutboxDeliveryView{}, err
	}
	return assembler.ToOutboxDeliveryView(delivery), nil
}

func (s *OutboxService) recordEvent(
	ctx context.Context,
	delivery model.OutboxDelivery,
	eventType model.OutboxDeliveryEventType,
	timestamp time.Time,
) error {
	if s == nil || s.events == nil {
		return nil
	}
	event, err := model.NewOutboxDeliveryEventFromDelivery(
		outboxDeliveryEventID(delivery.Message.EventID, eventType, timestamp),
		eventType,
		delivery,
		timestamp,
	)
	if err != nil {
		return err
	}
	return s.events.AppendOutboxDeliveryEvent(ctx, event)
}

func outboxDeliveryEventID(
	deliveryID string,
	eventType model.OutboxDeliveryEventType,
	timestamp time.Time,
) string {
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	return fmt.Sprintf(
		"outbox-event:%s:%s:%d",
		deliveryID,
		eventType,
		timestamp.UTC().UnixNano(),
	)
}

func (s *OutboxService) getModel(ctx context.Context, eventID string) (model.OutboxDelivery, error) {
	if s == nil || s.repository == nil {
		return model.OutboxDelivery{}, errors.New("outbox service requires repository")
	}
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return model.OutboxDelivery{}, errors.New("outbox event id required")
	}
	delivery, ok, err := s.repository.FindOutboxDelivery(ctx, eventID)
	if err != nil {
		return model.OutboxDelivery{}, err
	}
	if !ok {
		return model.OutboxDelivery{}, errors.New("outbox delivery not found")
	}
	return delivery, nil
}
