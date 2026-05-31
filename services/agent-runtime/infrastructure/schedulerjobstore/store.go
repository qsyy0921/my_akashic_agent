package schedulerjobstore

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

const storeVersion = "2026-05-31.schedulerjobstore.v1"

type Store struct {
	mu    sync.Mutex
	path  string
	jobs  map[string]model.SchedulerJob
	order []string
}

type persistedState struct {
	Version string               `json:"version"`
	Jobs    []model.SchedulerJob `json:"jobs"`
}

func NewStore(path string) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("scheduler job store path is required")
	}
	cleanPath := filepath.Clean(path)
	store := &Store{
		path:  cleanPath,
		jobs:  make(map[string]model.SchedulerJob),
		order: make([]string, 0),
	}
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
		return nil, err
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) ReplaceSchedulerJobs(_ context.Context, jobs []model.SchedulerJob) error {
	nextJobs := make(map[string]model.SchedulerJob, len(jobs))
	nextOrder := make([]string, 0, len(jobs))
	for _, job := range jobs {
		if err := job.Validate(); err != nil {
			return err
		}
		if _, exists := nextJobs[job.ID]; exists {
			continue
		}
		nextJobs[job.ID] = job
		nextOrder = append(nextOrder, job.ID)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs = nextJobs
	s.order = nextOrder
	return s.flush()
}

func (s *Store) UpsertSchedulerJob(_ context.Context, job model.SchedulerJob) (bool, error) {
	if s == nil {
		return false, errors.New("scheduler job store is nil")
	}
	if err := job.Validate(); err != nil {
		return false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.jobs[job.ID]
	s.jobs[job.ID] = job
	if !exists {
		s.order = append(s.order, job.ID)
	}
	return !exists, s.flush()
}

func (s *Store) DeleteSchedulerJob(_ context.Context, jobID string) (bool, error) {
	if s == nil {
		return false, errors.New("scheduler job store is nil")
	}
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return false, errors.New("scheduler job delete requires id")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.jobs[jobID]; !exists {
		return false, s.flush()
	}
	delete(s.jobs, jobID)
	s.order = removeSchedulerJobID(s.order, jobID)
	return true, s.flush()
}

func (s *Store) ListSchedulerJobs(_ context.Context) ([]model.SchedulerJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]model.SchedulerJob, 0, len(s.order))
	for _, jobID := range s.order {
		if job, ok := s.jobs[jobID]; ok {
			items = append(items, job)
		}
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
	for _, job := range state.Jobs {
		if err := job.Validate(); err != nil {
			continue
		}
		if _, exists := s.jobs[job.ID]; exists {
			continue
		}
		s.jobs[job.ID] = job
		s.order = append(s.order, job.ID)
	}
	return nil
}

func (s *Store) flush() error {
	state := persistedState{
		Version: storeVersion,
		Jobs:    make([]model.SchedulerJob, 0, len(s.order)),
	}
	for _, jobID := range s.order {
		if job, ok := s.jobs[jobID]; ok {
			state.Jobs = append(state.Jobs, job)
		}
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

func removeSchedulerJobID(items []string, jobID string) []string {
	result := items[:0]
	for _, item := range items {
		if item != jobID {
			result = append(result, item)
		}
	}
	return result
}
