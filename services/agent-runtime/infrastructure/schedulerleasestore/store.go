package schedulerleasestore

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

const storeVersion = "2026-05-31.schedulerleasestore.v1"

type Store struct {
	mu     sync.Mutex
	path   string
	leases map[string]model.SchedulerExecutionLease
}

type persistedState struct {
	Version string                          `json:"version"`
	Leases  []model.SchedulerExecutionLease `json:"leases"`
}

func NewStore(path string) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("scheduler lease store path is required")
	}
	cleanPath := filepath.Clean(path)
	store := &Store{
		path:   cleanPath,
		leases: make(map[string]model.SchedulerExecutionLease),
	}
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
		return nil, err
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) SaveSchedulerExecutionLease(_ context.Context, lease model.SchedulerExecutionLease) error {
	if s == nil {
		return errors.New("scheduler lease store is nil")
	}
	if err := lease.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.leases[lease.JobID] = lease
	return s.flush()
}

func (s *Store) DeleteSchedulerExecutionLease(_ context.Context, jobID string) error {
	if s == nil {
		return errors.New("scheduler lease store is nil")
	}
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return errors.New("scheduler execution lease delete requires job_id")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.leases, jobID)
	return s.flush()
}

func (s *Store) ListSchedulerExecutionLeases(_ context.Context) ([]model.SchedulerExecutionLease, error) {
	if s == nil {
		return nil, errors.New("scheduler lease store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return schedulerExecutionLeaseItems(s.leases), nil
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
	for _, lease := range state.Leases {
		if err := lease.Validate(); err != nil {
			continue
		}
		s.leases[lease.JobID] = lease
	}
	return nil
}

func (s *Store) flush() error {
	state := persistedState{
		Version: storeVersion,
		Leases:  schedulerExecutionLeaseItems(s.leases),
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

func schedulerExecutionLeaseItems(items map[string]model.SchedulerExecutionLease) []model.SchedulerExecutionLease {
	result := make([]model.SchedulerExecutionLease, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	return model.SortedSchedulerExecutionLeases(result)
}
