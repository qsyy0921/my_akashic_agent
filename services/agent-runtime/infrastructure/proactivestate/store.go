package proactivestate

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

const storeVersion = "2026-05-30.proactivestate.v1"

// Store persists deterministic proactive scheduling state for local runtime use.
type Store struct {
	mu            sync.Mutex
	path          string
	deliveries    map[string]model.ProactiveDeliveryRecord
	deliveryOrder []string
	contextOnly   []model.ProactiveContextOnlyRecord
	sessionMarks  map[string]model.ProactiveSessionMark
	anyAction     map[string]model.ProactiveAnyActionQuota
}

type persistedState struct {
	Version      string                             `json:"version"`
	Deliveries   []model.ProactiveDeliveryRecord    `json:"deliveries"`
	ContextOnly  []model.ProactiveContextOnlyRecord `json:"context_only"`
	SessionMarks []model.ProactiveSessionMark       `json:"session_marks"`
	AnyAction    []model.ProactiveAnyActionQuota    `json:"anyaction_quotas,omitempty"`
}

func NewStore(path string) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("proactive state store path is required")
	}
	cleanPath := filepath.Clean(path)
	store := &Store{
		path:         cleanPath,
		deliveries:   make(map[string]model.ProactiveDeliveryRecord),
		contextOnly:  make([]model.ProactiveContextOnlyRecord, 0),
		sessionMarks: make(map[string]model.ProactiveSessionMark),
		anyAction:    make(map[string]model.ProactiveAnyActionQuota),
	}
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
		return nil, err
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) SaveProactiveDelivery(_ context.Context, record model.ProactiveDeliveryRecord) error {
	if err := record.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := deliveryKey(record.SessionKey, record.DeliveryKey)
	if _, exists := s.deliveries[key]; !exists {
		s.deliveryOrder = append(s.deliveryOrder, key)
	}
	s.deliveries[key] = record
	return s.flush()
}

func (s *Store) FindProactiveDelivery(_ context.Context, sessionKey string, deliveryKeyValue string) (model.ProactiveDeliveryRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.deliveries[deliveryKey(sessionKey, deliveryKeyValue)]
	return record, ok, nil
}

func (s *Store) CountProactiveDeliveriesSince(_ context.Context, sessionKey string, since time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for _, record := range s.deliveries {
		if record.SessionKey == sessionKey && !record.SentAt.Before(since) {
			count++
		}
	}
	return count, nil
}

func (s *Store) ListProactiveDeliveries(_ context.Context, filter query.ProactiveDeliveryFilter) ([]model.ProactiveDeliveryRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	items := make([]model.ProactiveDeliveryRecord, 0, limit)
	for i := len(s.deliveryOrder) - 1; i >= 0 && len(items) < limit; i-- {
		key := s.deliveryOrder[i]
		record, ok := s.deliveries[key]
		if !ok {
			continue
		}
		if filter.SessionKey != "" && record.SessionKey != filter.SessionKey {
			continue
		}
		if filter.DeliveryKey != "" && record.DeliveryKey != filter.DeliveryKey {
			continue
		}
		items = append(items, record)
	}
	return items, nil
}

func (s *Store) SaveProactiveContextOnly(_ context.Context, record model.ProactiveContextOnlyRecord) error {
	if err := record.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.contextOnly = append(s.contextOnly, record)
	return s.flush()
}

func (s *Store) CountProactiveContextOnlySince(_ context.Context, sessionKey string, since time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for _, record := range s.contextOnly {
		if record.SessionKey == sessionKey && !record.SentAt.Before(since) {
			count++
		}
	}
	return count, nil
}

func (s *Store) SaveProactiveSessionMark(_ context.Context, mark model.ProactiveSessionMark) error {
	if err := mark.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessionMarks[sessionMarkKey(mark.SessionKey, mark.Key)] = mark
	return s.flush()
}

func (s *Store) FindProactiveSessionMark(_ context.Context, sessionKey string, key string) (model.ProactiveSessionMark, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	mark, ok := s.sessionMarks[sessionMarkKey(sessionKey, key)]
	return mark, ok, nil
}

func (s *Store) SaveProactiveAnyActionQuota(_ context.Context, quota model.ProactiveAnyActionQuota) error {
	if err := quota.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.anyAction[quota.QuotaKey] = quota
	return s.flush()
}

func (s *Store) FindProactiveAnyActionQuota(_ context.Context, quotaKey string) (model.ProactiveAnyActionQuota, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	quota, ok := s.anyAction[strings.TrimSpace(quotaKey)]
	return quota, ok, nil
}

func (s *Store) load() error {
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil
	}

	var state persistedState
	if err := json.Unmarshal(raw, &state); err != nil {
		return err
	}
	for _, record := range state.Deliveries {
		if err := record.Validate(); err != nil {
			continue
		}
		key := deliveryKey(record.SessionKey, record.DeliveryKey)
		if _, exists := s.deliveries[key]; !exists {
			s.deliveryOrder = append(s.deliveryOrder, key)
		}
		s.deliveries[key] = record
	}
	for _, record := range state.ContextOnly {
		if err := record.Validate(); err != nil {
			continue
		}
		s.contextOnly = append(s.contextOnly, record)
	}
	for _, mark := range state.SessionMarks {
		if err := mark.Validate(); err != nil {
			continue
		}
		s.sessionMarks[sessionMarkKey(mark.SessionKey, mark.Key)] = mark
	}
	for _, quota := range state.AnyAction {
		if err := quota.Validate(); err != nil {
			continue
		}
		s.anyAction[quota.QuotaKey] = quota
	}
	return nil
}

func (s *Store) flush() error {
	state := persistedState{
		Version:      storeVersion,
		Deliveries:   make([]model.ProactiveDeliveryRecord, 0, len(s.deliveryOrder)),
		ContextOnly:  append(make([]model.ProactiveContextOnlyRecord, 0, len(s.contextOnly)), s.contextOnly...),
		SessionMarks: make([]model.ProactiveSessionMark, 0, len(s.sessionMarks)),
		AnyAction:    make([]model.ProactiveAnyActionQuota, 0, len(s.anyAction)),
	}
	for _, key := range s.deliveryOrder {
		if record, ok := s.deliveries[key]; ok {
			state.Deliveries = append(state.Deliveries, record)
		}
	}
	for _, mark := range s.sessionMarks {
		state.SessionMarks = append(state.SessionMarks, mark)
	}
	for _, quota := range s.anyAction {
		state.AnyAction = append(state.AnyAction, quota)
	}

	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o600); err != nil {
		return err
	}
	if _, err := os.Stat(s.path); err == nil {
		if err := os.Remove(s.path); err != nil {
			return err
		}
	}
	return os.Rename(tmpPath, s.path)
}

func deliveryKey(sessionKey string, deliveryKeyValue string) string {
	return strings.TrimSpace(sessionKey) + "\x00" + strings.TrimSpace(deliveryKeyValue)
}

func sessionMarkKey(sessionKey string, key string) string {
	return strings.TrimSpace(sessionKey) + "\x00" + strings.TrimSpace(key)
}
