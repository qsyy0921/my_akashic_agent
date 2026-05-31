package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

const (
	defaultInboundDedupeTTLSeconds = 24 * 60 * 60
	minInboundDedupeTTLSeconds     = 30
	maxInboundDedupeTTLSeconds     = 30 * 24 * 60 * 60
	defaultInboundDedupeLimit      = 100
	maxInboundDedupeLimit          = 1000
)

type InboundDedupeService struct {
	repository outport.InboundDedupeRepository
}

func NewInboundDedupeService(repository outport.InboundDedupeRepository) *InboundDedupeService {
	return &InboundDedupeService{repository: repository}
}

func (s *InboundDedupeService) Check(ctx context.Context, cmd command.CheckInboundDedupeCommand) (query.InboundDedupeView, error) {
	if err := ctx.Err(); err != nil {
		return query.InboundDedupeView{}, err
	}
	if s == nil || s.repository == nil {
		return query.InboundDedupeView{}, errors.New("inbound dedupe service requires repository")
	}
	scope := strings.TrimSpace(cmd.Scope)
	messageKey := strings.TrimSpace(cmd.MessageKey)
	if scope == "" {
		return query.InboundDedupeView{}, errors.New("inbound dedupe requires scope")
	}
	if messageKey == "" {
		return query.InboundDedupeView{}, errors.New("inbound dedupe requires message_key")
	}
	now := cmd.Timestamp
	if now.IsZero() {
		now = time.Now().UTC()
	}
	ttl := inboundDedupeTTL(cmd.TTLSeconds)
	if _, err := s.repository.DeleteExpiredInboundDedupeRecords(ctx, now); err != nil {
		return query.InboundDedupeView{}, err
	}
	if existing, ok, err := s.repository.FindInboundDedupeRecord(ctx, scope, messageKey); err != nil {
		return query.InboundDedupeView{}, err
	} else if ok && existing.ActiveAt(now) {
		record, err := existing.MarkSeen(now)
		if err != nil {
			return query.InboundDedupeView{}, err
		}
		if err := s.repository.SaveInboundDedupeRecord(ctx, record); err != nil {
			return query.InboundDedupeView{}, err
		}
		return inboundDedupeView(record, true, ttl), nil
	}

	record, err := model.NewInboundDedupeRecord(model.InboundDedupeSpec{
		Scope:      scope,
		MessageKey: messageKey,
		SeenAt:     now,
		ExpiresAt:  now.Add(ttl),
		Metadata:   cmd.Metadata,
	})
	if err != nil {
		return query.InboundDedupeView{}, err
	}
	if err := s.repository.SaveInboundDedupeRecord(ctx, record); err != nil {
		return query.InboundDedupeView{}, err
	}
	return inboundDedupeView(record, false, ttl), nil
}

func (s *InboundDedupeService) List(ctx context.Context, filter query.InboundDedupeFilter) (query.InboundDedupeRecordsView, error) {
	if err := ctx.Err(); err != nil {
		return query.InboundDedupeRecordsView{}, err
	}
	if s == nil || s.repository == nil {
		return query.InboundDedupeRecordsView{}, errors.New("inbound dedupe service requires repository")
	}
	items, err := s.repository.ListInboundDedupeRecords(ctx, query.InboundDedupeFilter{
		Limit: boundedInboundDedupeLimit(filter.Limit),
		Scope: filter.Scope,
	})
	if err != nil {
		return query.InboundDedupeRecordsView{}, err
	}
	records := make([]query.InboundDedupeView, 0, len(items))
	for _, item := range items {
		records = append(records, inboundDedupeView(item, true, item.ExpiresAt.Sub(item.FirstSeen)))
	}
	return query.InboundDedupeRecordsView{
		Records: records,
		Totals: map[string]int{
			"records": len(records),
		},
		Notes:      []string{"side_effect=runtime_state_only"},
		SideEffect: "runtime_state_only",
	}, nil
}

func inboundDedupeView(record model.InboundDedupeRecord, duplicate bool, ttl time.Duration) query.InboundDedupeView {
	return query.InboundDedupeView{
		Duplicate:   duplicate,
		Scope:       record.Scope,
		MessageKey:  record.MessageKey,
		FirstSeenAt: record.FirstSeen.UTC().Format(time.RFC3339Nano),
		LastSeenAt:  record.LastSeen.UTC().Format(time.RFC3339Nano),
		ExpiresAt:   record.ExpiresAt.UTC().Format(time.RFC3339Nano),
		SeenCount:   record.SeenCount,
		TTLSeconds:  int(ttl.Seconds()),
		Metadata:    record.Metadata,
		SideEffect:  "runtime_state_only",
	}
}

func inboundDedupeTTL(value int) time.Duration {
	if value <= 0 {
		value = defaultInboundDedupeTTLSeconds
	}
	if value < minInboundDedupeTTLSeconds {
		value = minInboundDedupeTTLSeconds
	}
	if value > maxInboundDedupeTTLSeconds {
		value = maxInboundDedupeTTLSeconds
	}
	return time.Duration(value) * time.Second
}

func boundedInboundDedupeLimit(value int) int {
	if value <= 0 {
		return defaultInboundDedupeLimit
	}
	if value > maxInboundDedupeLimit {
		return maxInboundDedupeLimit
	}
	return value
}
