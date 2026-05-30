package model

import (
	"errors"
	"strings"
	"time"
)

type InboxEvent struct {
	Envelope   MessageEnvelope
	Decision   LoopDecision
	ReceivedAt time.Time
}

func NewInboxEvent(envelope MessageEnvelope, decision LoopDecision, receivedAt time.Time) (InboxEvent, error) {
	if receivedAt.IsZero() {
		receivedAt = time.Now().UTC()
	}
	event := InboxEvent{
		Envelope:   envelope,
		Decision:   decision,
		ReceivedAt: receivedAt.UTC(),
	}
	return event, event.Validate()
}

func (e InboxEvent) Validate() error {
	if err := e.Envelope.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(string(e.Decision.Action)) == "" {
		return errors.New("inbox event requires decision action")
	}
	if e.ReceivedAt.IsZero() {
		return errors.New("inbox event requires received_at")
	}
	return nil
}

func (e InboxEvent) EventID() string {
	return e.Envelope.EventID
}

func (e InboxEvent) ObserveOnly() bool {
	value := strings.ToLower(strings.TrimSpace(e.Envelope.Metadata["observe_only"]))
	return value == "true" || value == "1" || value == "yes" || e.Decision.Action == LoopActionObserveOnly
}
