package model

import (
	"errors"
	"sort"
	"strings"
	"time"
)

type InboundDedupeRecord struct {
	Scope      string            `json:"scope"`
	MessageKey string            `json:"message_key"`
	FirstSeen  time.Time         `json:"first_seen_at"`
	LastSeen   time.Time         `json:"last_seen_at"`
	ExpiresAt  time.Time         `json:"expires_at"`
	SeenCount  int               `json:"seen_count"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type InboundDedupeSpec struct {
	Scope      string
	MessageKey string
	SeenAt     time.Time
	ExpiresAt  time.Time
	Metadata   map[string]string
}

func NewInboundDedupeRecord(spec InboundDedupeSpec) (InboundDedupeRecord, error) {
	seenAt := spec.SeenAt
	if seenAt.IsZero() {
		seenAt = time.Now().UTC()
	}
	record := InboundDedupeRecord{
		Scope:      strings.TrimSpace(spec.Scope),
		MessageKey: strings.TrimSpace(spec.MessageKey),
		FirstSeen:  seenAt.UTC(),
		LastSeen:   seenAt.UTC(),
		ExpiresAt:  spec.ExpiresAt.UTC(),
		SeenCount:  1,
		Metadata:   cloneStringMap(spec.Metadata),
	}
	return record, record.Validate()
}

func (r InboundDedupeRecord) Validate() error {
	if strings.TrimSpace(r.Scope) == "" {
		return errors.New("inbound dedupe requires scope")
	}
	if strings.TrimSpace(r.MessageKey) == "" {
		return errors.New("inbound dedupe requires message_key")
	}
	if r.FirstSeen.IsZero() {
		return errors.New("inbound dedupe requires first_seen_at")
	}
	if r.LastSeen.IsZero() {
		return errors.New("inbound dedupe requires last_seen_at")
	}
	if r.ExpiresAt.IsZero() {
		return errors.New("inbound dedupe requires expires_at")
	}
	if r.SeenCount <= 0 {
		return errors.New("inbound dedupe requires seen_count")
	}
	return nil
}

func (r InboundDedupeRecord) ActiveAt(now time.Time) bool {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return r.ExpiresAt.After(now.UTC())
}

func (r InboundDedupeRecord) MarkSeen(seenAt time.Time) (InboundDedupeRecord, error) {
	if seenAt.IsZero() {
		seenAt = time.Now().UTC()
	}
	next := r
	next.LastSeen = seenAt.UTC()
	next.SeenCount++
	return next, next.Validate()
}

func SortedInboundDedupeRecords(items []InboundDedupeRecord) []InboundDedupeRecord {
	sorted := append([]InboundDedupeRecord(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if !sorted[i].LastSeen.Equal(sorted[j].LastSeen) {
			return sorted[i].LastSeen.After(sorted[j].LastSeen)
		}
		if sorted[i].Scope != sorted[j].Scope {
			return sorted[i].Scope < sorted[j].Scope
		}
		return sorted[i].MessageKey < sorted[j].MessageKey
	})
	return sorted
}
