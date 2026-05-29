package agentjobstore

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/domain/model"
)

const storeVersion = "2026-05-30.agentjobstore.v1"

// Store persists agent jobs for Go-owned lifecycle state.
type Store struct {
	mu    sync.Mutex
	path  string
	jobs  map[string]model.AgentJob
	order []string
}

type persistedState struct {
	Version string           `json:"version"`
	Jobs    []model.AgentJob `json:"jobs"`
}

func NewStore(path string) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("agent job store path is required")
	}
	cleanPath := filepath.Clean(path)
	store := &Store{
		path:  cleanPath,
		jobs:  make(map[string]model.AgentJob),
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

func (s *Store) SaveAgentJob(_ context.Context, job model.AgentJob) error {
	if err := job.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[job.JobID]; !exists {
		s.order = append(s.order, job.JobID)
	}
	s.jobs[job.JobID] = job
	return s.flush()
}

func (s *Store) FindAgentJob(_ context.Context, jobID string) (model.AgentJob, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return model.AgentJob{}, false, nil
	}
	return job, true, nil
}

func (s *Store) ListAgentJobs(_ context.Context, filter query.AgentJobFilter) ([]model.AgentJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	items := make([]model.AgentJob, 0, limit)
	for i := len(s.order) - 1; i >= 0 && len(items) < limit; i-- {
		jobID := s.order[i]
		job, ok := s.jobs[jobID]
		if !ok {
			continue
		}
		if filter.JobType != "" && string(job.JobType) != filter.JobType {
			continue
		}
		if filter.Status != "" && string(job.Status) != filter.Status {
			continue
		}
		items = append(items, job)
	}
	return items, nil
}

func (s *Store) FindLeaseableAgentJob(_ context.Context, jobType string, now time.Time) (model.AgentJob, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if now.IsZero() {
		now = time.Now().UTC()
	}
	for _, jobID := range s.order {
		job, ok := s.jobs[jobID]
		if !ok {
			continue
		}
		if jobType != "" && string(job.JobType) != jobType {
			continue
		}
		if job.CanLease(now) {
			return job, true, nil
		}
	}
	return model.AgentJob{}, false, nil
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
		if job.JobID == "" {
			continue
		}
		if err := job.Validate(); err != nil {
			continue
		}
		s.jobs[job.JobID] = job
		s.order = append(s.order, job.JobID)
	}
	return nil
}

func (s *Store) flush() error {
	state := persistedState{
		Version: storeVersion,
		Jobs:    make([]model.AgentJob, 0, len(s.order)),
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
