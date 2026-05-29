package sendledgerstore

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

const storeVersion = "2026-05-30.sendledgerstore.v1"

// Store persists outbound send records used by loop guard echo detection.
type Store struct {
	mu      sync.Mutex
	path    string
	records []model.SendRecord
}

type persistedState struct {
	Version string             `json:"version"`
	Records []model.SendRecord `json:"records"`
}

func NewStore(path string) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("send ledger store path is required")
	}
	cleanPath := filepath.Clean(path)
	store := &Store{
		path:    cleanPath,
		records: make([]model.SendRecord, 0),
	}
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
		return nil, err
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) RecordSent(_ context.Context, record model.SendRecord) error {
	if err := record.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.records = append(s.records, record)
	return s.flush()
}

func (s *Store) RecentlySent(botID string, conversationID string, contentHash string, window time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-window)
	for i := len(s.records) - 1; i >= 0; i-- {
		record := s.records[i]
		if record.Timestamp.Before(cutoff) {
			continue
		}
		if record.FromBotID == botID && record.ConversationID == conversationID && record.ContentHash == contentHash {
			return true
		}
	}
	return false
}

func (s *Store) ListSentRecords(_ context.Context, filter query.SendRecordFilter) ([]model.SendRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	items := make([]model.SendRecord, 0, limit)
	for i := len(s.records) - 1; i >= 0 && len(items) < limit; i-- {
		record := s.records[i]
		if filter.FromBotID != "" && record.FromBotID != filter.FromBotID {
			continue
		}
		if filter.ConversationID != "" && record.ConversationID != filter.ConversationID {
			continue
		}
		if filter.ContentHash != "" && record.ContentHash != filter.ContentHash {
			continue
		}
		items = append(items, record)
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
		s.records = append(s.records, record)
	}
	return nil
}

func (s *Store) flush() error {
	state := persistedState{
		Version: storeVersion,
		Records: append(
			make([]model.SendRecord, 0, len(s.records)),
			s.records...,
		),
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
