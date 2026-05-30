package model

import (
	"errors"
	"strings"
	"time"
)

type OutboxDeliveryEventType string

const (
	OutboxDeliveryEventQueued      OutboxDeliveryEventType = "queued"
	OutboxDeliveryEventLeased      OutboxDeliveryEventType = "leased"
	OutboxDeliveryEventDispatching OutboxDeliveryEventType = "dispatching"
	OutboxDeliveryEventSucceeded   OutboxDeliveryEventType = "succeeded"
	OutboxDeliveryEventFailed      OutboxDeliveryEventType = "failed"
	OutboxDeliveryEventRetry       OutboxDeliveryEventType = "retry"
)

type OutboxDeliveryEvent struct {
	EventID        string
	DeliveryID     string
	Channel        ChannelRef
	EventType      OutboxDeliveryEventType
	Status         DeliveryStatus
	Attempt        int
	MaxAttempts    int
	LeaseOwner     string
	LeaseExpiresAt time.Time
	ErrorKind      DeliveryErrorKind
	ErrorMessage   string
	OccurredAt     time.Time
	Metadata       map[string]string
}

type OutboxDeliveryEventSpec struct {
	EventID        string
	DeliveryID     string
	Channel        ChannelRef
	EventType      OutboxDeliveryEventType
	Status         DeliveryStatus
	Attempt        int
	MaxAttempts    int
	LeaseOwner     string
	LeaseExpiresAt time.Time
	ErrorKind      DeliveryErrorKind
	ErrorMessage   string
	Metadata       map[string]string
}

func NewOutboxDeliveryEvent(spec OutboxDeliveryEventSpec, now time.Time) (OutboxDeliveryEvent, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	errorKind := DeliveryErrorKind("")
	if strings.TrimSpace(string(spec.ErrorKind)) != "" {
		errorKind = NormalizeDeliveryErrorKind(string(spec.ErrorKind))
	}
	event := OutboxDeliveryEvent{
		EventID:        strings.TrimSpace(spec.EventID),
		DeliveryID:     strings.TrimSpace(spec.DeliveryID),
		Channel:        spec.Channel,
		EventType:      spec.EventType,
		Status:         spec.Status,
		Attempt:        spec.Attempt,
		MaxAttempts:    spec.MaxAttempts,
		LeaseOwner:     strings.TrimSpace(spec.LeaseOwner),
		LeaseExpiresAt: spec.LeaseExpiresAt,
		ErrorKind:      errorKind,
		ErrorMessage:   strings.TrimSpace(spec.ErrorMessage),
		OccurredAt:     now,
		Metadata:       copyStringMap(spec.Metadata),
	}
	if err := event.Validate(); err != nil {
		return OutboxDeliveryEvent{}, err
	}
	return event, nil
}

func NewOutboxDeliveryEventFromDelivery(
	eventID string,
	eventType OutboxDeliveryEventType,
	delivery OutboxDelivery,
	now time.Time,
) (OutboxDeliveryEvent, error) {
	metadata := map[string]string{}
	if delivery.ErrorMessage != "" {
		metadata["error_message"] = delivery.ErrorMessage
	}
	if delivery.ErrorKind != "" {
		metadata["error_kind"] = string(delivery.ErrorKind)
	}
	return NewOutboxDeliveryEvent(OutboxDeliveryEventSpec{
		EventID:        eventID,
		DeliveryID:     delivery.Message.EventID,
		Channel:        delivery.Message.Channel,
		EventType:      eventType,
		Status:         delivery.Status,
		Attempt:        delivery.Attempts,
		MaxAttempts:    delivery.MaxAttempts,
		LeaseOwner:     delivery.LeaseOwner,
		LeaseExpiresAt: delivery.LeaseExpiresAt,
		ErrorKind:      delivery.ErrorKind,
		ErrorMessage:   delivery.ErrorMessage,
		Metadata:       metadata,
	}, now)
}

func (e OutboxDeliveryEvent) Validate() error {
	if strings.TrimSpace(e.EventID) == "" {
		return errors.New("outbox delivery event requires event id")
	}
	if strings.TrimSpace(e.DeliveryID) == "" {
		return errors.New("outbox delivery event requires delivery id")
	}
	if strings.TrimSpace(string(e.Channel.Kind)) == "" {
		return errors.New("outbox delivery event requires channel kind")
	}
	if strings.TrimSpace(e.Channel.AccountID) == "" {
		return errors.New("outbox delivery event requires account id")
	}
	if strings.TrimSpace(e.Channel.ConversationID) == "" {
		return errors.New("outbox delivery event requires conversation id")
	}
	if strings.TrimSpace(string(e.Channel.ConversationType)) == "" {
		return errors.New("outbox delivery event requires conversation type")
	}
	switch e.EventType {
	case OutboxDeliveryEventQueued, OutboxDeliveryEventLeased, OutboxDeliveryEventDispatching, OutboxDeliveryEventSucceeded, OutboxDeliveryEventFailed, OutboxDeliveryEventRetry:
	default:
		return errors.New("outbox delivery event has invalid event type")
	}
	switch e.Status {
	case DeliveryQueued, DeliveryDispatching, DeliverySucceeded, DeliveryFailed, DeliveryDeadLettered:
	default:
		return errors.New("outbox delivery event has invalid status")
	}
	if e.Attempt < 0 {
		return errors.New("outbox delivery event attempt cannot be negative")
	}
	if e.MaxAttempts <= 0 {
		return errors.New("outbox delivery event requires positive max attempts")
	}
	if e.OccurredAt.IsZero() {
		return errors.New("outbox delivery event requires occurred_at")
	}
	if e.ErrorKind != "" && NormalizeDeliveryErrorKind(string(e.ErrorKind)) != e.ErrorKind {
		return errors.New("outbox delivery event has invalid error kind")
	}
	return nil
}
