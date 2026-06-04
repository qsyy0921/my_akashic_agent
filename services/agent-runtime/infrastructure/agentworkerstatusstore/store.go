package agentworkerstatusstore

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

const storeVersion = "2026-05-31.agentworkerstatusstore.v1"

type Store struct {
	mu      sync.Mutex
	path    string
	workers map[string]model.AgentWorkerStatus
}

type persistedState struct {
	Version string                    `json:"version"`
	Workers []model.AgentWorkerStatus `json:"workers"`
}

func NewStore(path string) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("agent worker status store path is required")
	}
	cleanPath := filepath.Clean(path)
	store := &Store{
		path:    cleanPath,
		workers: make(map[string]model.AgentWorkerStatus),
	}
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
		return nil, err
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) SaveAgentWorkerStatus(_ context.Context, status model.AgentWorkerStatus) error {
	if s == nil {
		return errors.New("agent worker status store is nil")
	}
	if err := status.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.workers[status.WorkerID] = status
	return s.flush()
}

func (s *Store) DeleteAgentWorkerStatus(_ context.Context, workerID string) error {
	if s == nil {
		return errors.New("agent worker status store is nil")
	}
	workerID = strings.TrimSpace(workerID)
	if workerID == "" {
		return errors.New("agent worker status delete requires worker_id")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.workers, workerID)
	return s.flush()
}

func (s *Store) ListAgentWorkerStatuses(_ context.Context) ([]model.AgentWorkerStatus, error) {
	if s == nil {
		return nil, errors.New("agent worker status store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return model.SortedAgentWorkerStatuses(agentWorkerStatusItems(s.workers)), nil
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
	for _, status := range state.Workers {
		if err := status.Validate(); err != nil {
			continue
		}
		s.workers[status.WorkerID] = status
	}
	return nil
}

func (s *Store) flush() error {
	state := persistedState{
		Version: storeVersion,
		Workers: model.SortedAgentWorkerStatuses(
			agentWorkerStatusItems(s.workers),
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

func agentWorkerStatusItems(items map[string]model.AgentWorkerStatus) []model.AgentWorkerStatus {
	result := make([]model.AgentWorkerStatus, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	return result
}
