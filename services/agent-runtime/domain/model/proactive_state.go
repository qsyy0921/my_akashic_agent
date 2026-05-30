package model

import (
	"errors"
	"strings"
	"time"
)

const (
	ProactiveSessionMarkContextOnlyLastAt = "context_only_last_at"
	ProactiveSessionMarkDriftLastAt       = "drift_last_at"
)

type ProactiveDeliveryRecord struct {
	SessionKey  string
	DeliveryKey string
	SentAt      time.Time
}

func NewProactiveDeliveryRecord(sessionKey string, deliveryKey string, sentAt time.Time) (ProactiveDeliveryRecord, error) {
	if sentAt.IsZero() {
		sentAt = time.Now().UTC()
	}
	record := ProactiveDeliveryRecord{
		SessionKey:  strings.TrimSpace(sessionKey),
		DeliveryKey: strings.TrimSpace(deliveryKey),
		SentAt:      sentAt,
	}
	if err := record.Validate(); err != nil {
		return ProactiveDeliveryRecord{}, err
	}
	return record, nil
}

func (r ProactiveDeliveryRecord) Validate() error {
	if strings.TrimSpace(r.SessionKey) == "" {
		return errors.New("proactive delivery requires session key")
	}
	if strings.TrimSpace(r.DeliveryKey) == "" {
		return errors.New("proactive delivery requires delivery key")
	}
	if r.SentAt.IsZero() {
		return errors.New("proactive delivery requires sent_at")
	}
	return nil
}

func (r ProactiveDeliveryRecord) WithinWindow(now time.Time, window time.Duration) bool {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if window <= 0 {
		return false
	}
	return !r.SentAt.Before(now.Add(-window))
}

type ProactiveContextOnlyRecord struct {
	SessionKey string
	SentAt     time.Time
}

func NewProactiveContextOnlyRecord(sessionKey string, sentAt time.Time) (ProactiveContextOnlyRecord, error) {
	if sentAt.IsZero() {
		sentAt = time.Now().UTC()
	}
	record := ProactiveContextOnlyRecord{
		SessionKey: strings.TrimSpace(sessionKey),
		SentAt:     sentAt,
	}
	if err := record.Validate(); err != nil {
		return ProactiveContextOnlyRecord{}, err
	}
	return record, nil
}

func (r ProactiveContextOnlyRecord) Validate() error {
	if strings.TrimSpace(r.SessionKey) == "" {
		return errors.New("proactive context-only record requires session key")
	}
	if r.SentAt.IsZero() {
		return errors.New("proactive context-only record requires sent_at")
	}
	return nil
}

type ProactiveSessionMark struct {
	SessionKey string
	Key        string
	MarkedAt   time.Time
}

func NewProactiveSessionMark(sessionKey string, key string, markedAt time.Time) (ProactiveSessionMark, error) {
	if markedAt.IsZero() {
		markedAt = time.Now().UTC()
	}
	mark := ProactiveSessionMark{
		SessionKey: strings.TrimSpace(sessionKey),
		Key:        strings.TrimSpace(key),
		MarkedAt:   markedAt,
	}
	if err := mark.Validate(); err != nil {
		return ProactiveSessionMark{}, err
	}
	return mark, nil
}

func (m ProactiveSessionMark) Validate() error {
	if strings.TrimSpace(m.SessionKey) == "" {
		return errors.New("proactive session mark requires session key")
	}
	switch strings.TrimSpace(m.Key) {
	case ProactiveSessionMarkContextOnlyLastAt, ProactiveSessionMarkDriftLastAt:
	default:
		return errors.New("proactive session mark has invalid key")
	}
	if m.MarkedAt.IsZero() {
		return errors.New("proactive session mark requires marked_at")
	}
	return nil
}
