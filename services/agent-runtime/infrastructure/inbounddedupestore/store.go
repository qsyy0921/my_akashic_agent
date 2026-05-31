package inbounddedupestore

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

const storeVersion = "2026-05-31.inbounddedupestore.v1"

type Store struct {
	mu      sync.Mutex
	path    string
	records map[string]model.InboundDedupeRecord
}

type persistedState struct {
	Version string                      `json:"version"`
	Records []model.InboundDedupeRecord `json:"records"`
}

func NewStore(path string) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("inbound dedupe store path is required")
	}
	cleanPath := filepath.Clean(path)
	store := &Store{
		path:    cleanPath,
		records: make(map[string]model.InboundDedupeRecord),
	}
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
		return nil, err
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) FindInboundDedupeRecord(_ context.Context, scope string, messageKey string) (model.InboundDedupeRecord, bool, error) {
	if s == nil {
		return model.InboundDedupeRecord{}, false, errors.New("inbound dedupe store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[inboundDedupeRecordKey(scope, messageKey)]
	return record, ok, nil
}

func (s *Store) SaveInboundDedupeRecord(_ context.Context, record model.InboundDedupeRecord) error {
	if s == nil {
		return errors.New("inbound dedupe store is nil")
	}
	if err := record.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[inboundDedupeRecordKey(record.Scope, record.MessageKey)] = record
	return s.flush()
}

func (s *Store) DeleteExpiredInboundDedupeRecords(_ context.Context, now time.Time) (int, error) {
	if s == nil {
		return 0, errors.New("inbound dedupe store is nil")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	deleted := 0
	for key, record := range s.records {
		if !record.ActiveAt(now) {
			delete(s.records, key)
			deleted++
		}
	}
	if deleted == 0 {
		return 0, nil
	}
	return deleted, s.flush()
}

func (s *Store) ListInboundDedupeRecords(_ context.Context, filter query.InboundDedupeFilter) ([]model.InboundDedupeRecord, error) {
	if s == nil {
		return nil, errors.New("inbound dedupe store is nil")
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	scope := strings.TrimSpace(filter.Scope)
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]model.InboundDedupeRecord, 0, len(s.records))
	for _, record := range s.records {
		if scope != "" && record.Scope != scope {
			continue
		}
		items = append(items, record)
	}
	items = model.SortedInboundDedupeRecords(items)
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
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
	for _, record := range state.Records {
		if err := record.Validate(); err != nil {
			continue
		}
		s.records[inboundDedupeRecordKey(record.Scope, record.MessageKey)] = record
	}
	return nil
}

func (s *Store) flush() error {
	state := persistedState{
		Version: storeVersion,
		Records: model.SortedInboundDedupeRecords(s.recordItems()),
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

func (s *Store) recordItems() []model.InboundDedupeRecord {
	items := make([]model.InboundDedupeRecord, 0, len(s.records))
	for _, record := range s.records {
		items = append(items, record)
	}
	return items
}

func inboundDedupeRecordKey(scope string, messageKey string) string {
	return strings.TrimSpace(scope) + "\x00" + strings.TrimSpace(messageKey)
}
