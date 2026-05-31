package receiverstatusstore

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

const storeVersion = "2026-05-31.receiverstatusstore.v1"

type Store struct {
	mu        sync.Mutex
	path      string
	receivers map[string]model.ReceiverStatus
}

type persistedState struct {
	Version   string                 `json:"version"`
	Receivers []model.ReceiverStatus `json:"receivers"`
}

func NewStore(path string) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("receiver status store path is required")
	}
	cleanPath := filepath.Clean(path)
	store := &Store{
		path:      cleanPath,
		receivers: make(map[string]model.ReceiverStatus),
	}
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
		return nil, err
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) SaveReceiverStatus(_ context.Context, status model.ReceiverStatus) error {
	if s == nil {
		return errors.New("receiver status store is nil")
	}
	if err := status.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.receivers[status.ReceiverID] = status
	return s.flush()
}

func (s *Store) ListReceiverStatuses(_ context.Context) ([]model.ReceiverStatus, error) {
	if s == nil {
		return nil, errors.New("receiver status store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return model.SortedReceiverStatuses(receiverStatusItems(s.receivers)), nil
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
	for _, status := range state.Receivers {
		if err := status.Validate(); err != nil {
			continue
		}
		s.receivers[status.ReceiverID] = status
	}
	return nil
}

func (s *Store) flush() error {
	state := persistedState{
		Version:   storeVersion,
		Receivers: model.SortedReceiverStatuses(receiverStatusItems(s.receivers)),
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

func receiverStatusItems(items map[string]model.ReceiverStatus) []model.ReceiverStatus {
	result := make([]model.ReceiverStatus, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	return result
}
