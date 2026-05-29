package model

import (
	"errors"
	"strings"
	"time"
)

type DeliveryStatus string

const (
	DeliveryQueued       DeliveryStatus = "queued"
	DeliveryDispatching  DeliveryStatus = "dispatching"
	DeliverySucceeded    DeliveryStatus = "succeeded"
	DeliveryFailed       DeliveryStatus = "failed"
	DeliveryDeadLettered DeliveryStatus = "dead_lettered"
)

type OutboxDelivery struct {
	Message      OutboundMessage
	Status       DeliveryStatus
	Attempts     int
	MaxAttempts  int
	ErrorMessage string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewOutboxDelivery(message OutboundMessage, maxAttempts int, now time.Time) (OutboxDelivery, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	delivery := OutboxDelivery{
		Message:     message,
		Status:      DeliveryQueued,
		MaxAttempts: maxAttempts,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := delivery.Validate(); err != nil {
		return OutboxDelivery{}, err
	}
	return delivery, nil
}

func (d OutboxDelivery) Validate() error {
	if err := d.Message.Validate(); err != nil {
		return err
	}
	if d.Status == "" {
		return errors.New("outbox delivery requires status")
	}
	if d.Attempts < 0 {
		return errors.New("outbox delivery attempts cannot be negative")
	}
	if d.MaxAttempts <= 0 {
		return errors.New("outbox delivery requires positive max attempts")
	}
	if d.Attempts > d.MaxAttempts {
		return errors.New("outbox delivery attempts exceed max attempts")
	}
	if d.CreatedAt.IsZero() {
		return errors.New("outbox delivery requires created_at")
	}
	if d.UpdatedAt.IsZero() {
		return errors.New("outbox delivery requires updated_at")
	}
	switch d.Status {
	case DeliveryQueued, DeliveryDispatching, DeliverySucceeded, DeliveryFailed, DeliveryDeadLettered:
		return nil
	default:
		return errors.New("outbox delivery has invalid status")
	}
}

func (d *OutboxDelivery) MarkDispatching(now time.Time) error {
	if d == nil {
		return errors.New("outbox delivery is nil")
	}
	if d.Status == DeliverySucceeded {
		return errors.New("succeeded delivery cannot be dispatched again")
	}
	if d.Status == DeliveryDeadLettered {
		return errors.New("dead-lettered delivery cannot be dispatched")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if d.Attempts >= d.MaxAttempts {
		d.Status = DeliveryDeadLettered
		d.UpdatedAt = now
		return nil
	}
	d.Attempts++
	d.Status = DeliveryDispatching
	d.ErrorMessage = ""
	d.UpdatedAt = now
	return d.Validate()
}

func (d *OutboxDelivery) MarkSucceeded(now time.Time) error {
	if d == nil {
		return errors.New("outbox delivery is nil")
	}
	if d.Status == DeliveryDeadLettered {
		return errors.New("dead-lettered delivery cannot succeed")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	d.Status = DeliverySucceeded
	d.ErrorMessage = ""
	d.UpdatedAt = now
	return d.Validate()
}

func (d *OutboxDelivery) MarkFailed(message string, now time.Time) error {
	if d == nil {
		return errors.New("outbox delivery is nil")
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return errors.New("outbox delivery failure requires error message")
	}
	if d.Status == DeliverySucceeded {
		return errors.New("succeeded delivery cannot fail")
	}
	if d.Status == DeliveryDeadLettered {
		return errors.New("dead-lettered delivery cannot fail again")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if d.Attempts == 0 {
		d.Attempts = 1
	}
	d.ErrorMessage = message
	d.UpdatedAt = now
	if d.Attempts >= d.MaxAttempts {
		d.Status = DeliveryDeadLettered
		return d.Validate()
	}
	d.Status = DeliveryFailed
	return d.Validate()
}

func (d *OutboxDelivery) Retry(now time.Time) error {
	if d == nil {
		return errors.New("outbox delivery is nil")
	}
	if d.Status != DeliveryFailed {
		return errors.New("only failed delivery can be retried")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	d.Status = DeliveryQueued
	d.ErrorMessage = ""
	d.UpdatedAt = now
	return d.Validate()
}
