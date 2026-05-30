package model

import (
	"errors"
	"strings"
	"time"
)

type DeliveryStatus string
type DeliveryErrorKind string

const (
	DeliveryQueued       DeliveryStatus = "queued"
	DeliveryDispatching  DeliveryStatus = "dispatching"
	DeliverySucceeded    DeliveryStatus = "succeeded"
	DeliveryFailed       DeliveryStatus = "failed"
	DeliveryDeadLettered DeliveryStatus = "dead_lettered"
)

const (
	DeliveryErrorUnknown           DeliveryErrorKind = "unknown"
	DeliveryErrorPlatform          DeliveryErrorKind = "platform_error"
	DeliveryErrorPlatformTimeout   DeliveryErrorKind = "platform_timeout"
	DeliveryErrorRoute             DeliveryErrorKind = "route_error"
	DeliveryErrorUnsupportedMedia  DeliveryErrorKind = "unsupported_media"
	DeliveryErrorSenderUnavailable DeliveryErrorKind = "sender_unavailable"
	DeliveryErrorValidation        DeliveryErrorKind = "validation_error"
)

type OutboxDelivery struct {
	Message        OutboundMessage
	Status         DeliveryStatus
	Attempts       int
	MaxAttempts    int
	LeaseOwner     string
	LeaseExpiresAt time.Time
	ErrorKind      DeliveryErrorKind
	ErrorMessage   string
	CreatedAt      time.Time
	UpdatedAt      time.Time
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
	if !d.LeaseExpiresAt.IsZero() && strings.TrimSpace(d.LeaseOwner) == "" {
		return errors.New("outbox delivery lease expiry requires owner")
	}
	if d.ErrorKind != "" && NormalizeDeliveryErrorKind(string(d.ErrorKind)) != d.ErrorKind {
		return errors.New("outbox delivery has invalid error kind")
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
	d.LeaseOwner = ""
	d.LeaseExpiresAt = time.Time{}
	d.ErrorKind = ""
	d.ErrorMessage = ""
	d.UpdatedAt = now
	return d.Validate()
}

func (d OutboxDelivery) CanLease(now time.Time) bool {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	switch d.Status {
	case DeliveryQueued:
		return true
	case DeliveryDispatching:
		return !d.LeaseExpiresAt.IsZero() && !d.LeaseExpiresAt.After(now)
	default:
		return false
	}
}

func (d *OutboxDelivery) Lease(workerID string, ttl time.Duration, now time.Time) error {
	if d == nil {
		return errors.New("outbox delivery is nil")
	}
	workerID = strings.TrimSpace(workerID)
	if workerID == "" {
		return errors.New("outbox delivery lease requires worker id")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	if !d.CanLease(now) {
		return errors.New("outbox delivery is not leaseable")
	}
	if d.Attempts >= d.MaxAttempts {
		d.Status = DeliveryDeadLettered
		d.LeaseOwner = ""
		d.LeaseExpiresAt = time.Time{}
		d.UpdatedAt = now
		return d.Validate()
	}
	d.Attempts++
	d.Status = DeliveryDispatching
	d.LeaseOwner = workerID
	d.LeaseExpiresAt = now.Add(ttl)
	d.ErrorKind = ""
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
	d.LeaseOwner = ""
	d.LeaseExpiresAt = time.Time{}
	d.ErrorKind = ""
	d.ErrorMessage = ""
	d.UpdatedAt = now
	return d.Validate()
}

func (d *OutboxDelivery) MarkFailed(message string, now time.Time) error {
	return d.MarkFailedWithKind(DeliveryErrorUnknown, message, now)
}

func (d *OutboxDelivery) MarkFailedWithKind(kind DeliveryErrorKind, message string, now time.Time) error {
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
	d.LeaseOwner = ""
	d.LeaseExpiresAt = time.Time{}
	d.ErrorKind = NormalizeDeliveryErrorKind(string(kind))
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
	d.LeaseOwner = ""
	d.LeaseExpiresAt = time.Time{}
	d.ErrorKind = ""
	d.ErrorMessage = ""
	d.UpdatedAt = now
	return d.Validate()
}

func NormalizeDeliveryErrorKind(value string) DeliveryErrorKind {
	switch DeliveryErrorKind(strings.TrimSpace(value)) {
	case DeliveryErrorPlatform:
		return DeliveryErrorPlatform
	case DeliveryErrorPlatformTimeout:
		return DeliveryErrorPlatformTimeout
	case DeliveryErrorRoute:
		return DeliveryErrorRoute
	case DeliveryErrorUnsupportedMedia:
		return DeliveryErrorUnsupportedMedia
	case DeliveryErrorSenderUnavailable:
		return DeliveryErrorSenderUnavailable
	case DeliveryErrorValidation:
		return DeliveryErrorValidation
	case DeliveryErrorUnknown, "":
		return DeliveryErrorUnknown
	default:
		return DeliveryErrorUnknown
	}
}
