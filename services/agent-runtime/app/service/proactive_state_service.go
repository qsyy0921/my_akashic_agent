package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/assembler"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type ProactiveStateService struct {
	repository outport.ProactiveStateRepository
}

func NewProactiveStateService(repository outport.ProactiveStateRepository) *ProactiveStateService {
	return &ProactiveStateService{repository: repository}
}

func (s *ProactiveStateService) RecordDelivery(ctx context.Context, cmd command.RecordProactiveDeliveryCommand) (query.ProactiveDeliveryView, error) {
	if s == nil || s.repository == nil {
		return query.ProactiveDeliveryView{}, errors.New("proactive state service requires repository")
	}
	record, err := model.NewProactiveDeliveryRecord(cmd.SessionKey, cmd.DeliveryKey, timestampOrNow(cmd.Timestamp))
	if err != nil {
		return query.ProactiveDeliveryView{}, err
	}
	if err := s.repository.SaveProactiveDelivery(ctx, record); err != nil {
		return query.ProactiveDeliveryView{}, err
	}
	return assembler.ToProactiveDeliveryView(record), nil
}

func (s *ProactiveStateService) IsDeliveryDuplicate(ctx context.Context, cmd command.CheckProactiveDeliveryDuplicateCommand) (query.ProactiveDuplicateView, error) {
	if s == nil || s.repository == nil {
		return query.ProactiveDuplicateView{}, errors.New("proactive state service requires repository")
	}
	sessionKey := strings.TrimSpace(cmd.SessionKey)
	deliveryKey := strings.TrimSpace(cmd.DeliveryKey)
	if sessionKey == "" {
		return query.ProactiveDuplicateView{}, errors.New("session key required")
	}
	if deliveryKey == "" {
		return query.ProactiveDuplicateView{}, errors.New("delivery key required")
	}
	windowHours := positiveHoursOrDefault(cmd.WindowHours, 24)
	record, ok, err := s.repository.FindProactiveDelivery(ctx, sessionKey, deliveryKey)
	if err != nil {
		return query.ProactiveDuplicateView{}, err
	}
	duplicate := ok && record.WithinWindow(timestampOrNow(cmd.Timestamp), time.Duration(windowHours)*time.Hour)
	return query.ProactiveDuplicateView{
		Duplicate:   duplicate,
		SessionKey:  sessionKey,
		DeliveryKey: deliveryKey,
		WindowHours: windowHours,
	}, nil
}

func (s *ProactiveStateService) CountDeliveries(ctx context.Context, cmd command.CountProactiveDeliveriesCommand) (query.ProactiveCountView, error) {
	if s == nil || s.repository == nil {
		return query.ProactiveCountView{}, errors.New("proactive state service requires repository")
	}
	sessionKey := strings.TrimSpace(cmd.SessionKey)
	if sessionKey == "" {
		return query.ProactiveCountView{}, errors.New("session key required")
	}
	windowHours := positiveHoursOrDefault(cmd.WindowHours, 24)
	since := timestampOrNow(cmd.Timestamp).Add(-time.Duration(windowHours) * time.Hour)
	count, err := s.repository.CountProactiveDeliveriesSince(ctx, sessionKey, since)
	if err != nil {
		return query.ProactiveCountView{}, err
	}
	return query.ProactiveCountView{Count: count, SessionKey: sessionKey, WindowHours: windowHours}, nil
}

func (s *ProactiveStateService) ListDeliveries(ctx context.Context, filter query.ProactiveDeliveryFilter) ([]query.ProactiveDeliveryView, error) {
	if s == nil || s.repository == nil {
		return nil, errors.New("proactive state service requires repository")
	}
	items, err := s.repository.ListProactiveDeliveries(ctx, filter)
	if err != nil {
		return nil, err
	}
	return assembler.ToProactiveDeliveryViews(items), nil
}

func (s *ProactiveStateService) RecordContextOnly(ctx context.Context, cmd command.RecordProactiveContextOnlyCommand) (query.ProactiveTimestampView, error) {
	if s == nil || s.repository == nil {
		return query.ProactiveTimestampView{}, errors.New("proactive state service requires repository")
	}
	record, err := model.NewProactiveContextOnlyRecord(cmd.SessionKey, timestampOrNow(cmd.Timestamp))
	if err != nil {
		return query.ProactiveTimestampView{}, err
	}
	if err := s.repository.SaveProactiveContextOnly(ctx, record); err != nil {
		return query.ProactiveTimestampView{}, err
	}
	mark, err := model.NewProactiveSessionMark(record.SessionKey, model.ProactiveSessionMarkContextOnlyLastAt, record.SentAt)
	if err != nil {
		return query.ProactiveTimestampView{}, err
	}
	if err := s.repository.SaveProactiveSessionMark(ctx, mark); err != nil {
		return query.ProactiveTimestampView{}, err
	}
	return assembler.ToProactiveTimestampView(record.SessionKey, model.ProactiveSessionMarkContextOnlyLastAt, record.SentAt, true), nil
}

func (s *ProactiveStateService) LastContextOnly(ctx context.Context, sessionKey string) (query.ProactiveTimestampView, error) {
	return s.lastSessionMark(ctx, sessionKey, model.ProactiveSessionMarkContextOnlyLastAt)
}

func (s *ProactiveStateService) CountContextOnly(ctx context.Context, cmd command.CountProactiveContextOnlyCommand) (query.ProactiveCountView, error) {
	if s == nil || s.repository == nil {
		return query.ProactiveCountView{}, errors.New("proactive state service requires repository")
	}
	sessionKey := strings.TrimSpace(cmd.SessionKey)
	if sessionKey == "" {
		return query.ProactiveCountView{}, errors.New("session key required")
	}
	windowHours := positiveHoursOrDefault(cmd.WindowHours, 24)
	since := timestampOrNow(cmd.Timestamp).Add(-time.Duration(windowHours) * time.Hour)
	count, err := s.repository.CountProactiveContextOnlySince(ctx, sessionKey, since)
	if err != nil {
		return query.ProactiveCountView{}, err
	}
	return query.ProactiveCountView{Count: count, SessionKey: sessionKey, WindowHours: windowHours}, nil
}

func (s *ProactiveStateService) RecordDriftRun(ctx context.Context, cmd command.RecordProactiveDriftRunCommand) (query.ProactiveTimestampView, error) {
	if s == nil || s.repository == nil {
		return query.ProactiveTimestampView{}, errors.New("proactive state service requires repository")
	}
	mark, err := model.NewProactiveSessionMark(cmd.SessionKey, model.ProactiveSessionMarkDriftLastAt, timestampOrNow(cmd.Timestamp))
	if err != nil {
		return query.ProactiveTimestampView{}, err
	}
	if err := s.repository.SaveProactiveSessionMark(ctx, mark); err != nil {
		return query.ProactiveTimestampView{}, err
	}
	return assembler.ToProactiveTimestampView(mark.SessionKey, mark.Key, mark.MarkedAt, true), nil
}

func (s *ProactiveStateService) LastDriftRun(ctx context.Context, sessionKey string) (query.ProactiveTimestampView, error) {
	return s.lastSessionMark(ctx, sessionKey, model.ProactiveSessionMarkDriftLastAt)
}

func (s *ProactiveStateService) lastSessionMark(ctx context.Context, sessionKey string, key string) (query.ProactiveTimestampView, error) {
	if s == nil || s.repository == nil {
		return query.ProactiveTimestampView{}, errors.New("proactive state service requires repository")
	}
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return query.ProactiveTimestampView{}, errors.New("session key required")
	}
	mark, ok, err := s.repository.FindProactiveSessionMark(ctx, sessionKey, key)
	if err != nil {
		return query.ProactiveTimestampView{}, err
	}
	if !ok {
		return assembler.ToProactiveTimestampView(sessionKey, key, time.Time{}, false), nil
	}
	return assembler.ToProactiveTimestampView(mark.SessionKey, mark.Key, mark.MarkedAt, true), nil
}

func timestampOrNow(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value
}

func positiveHoursOrDefault(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}
