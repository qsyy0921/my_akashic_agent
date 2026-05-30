package knowledgecheckpointstore

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

const storeVersion = "2026-05-30.knowledgecheckpointstore.v1"

type Store struct {
	mu          sync.Mutex
	path        string
	checkpoints map[string]model.KnowledgeCheckpoint
	order       []string
}

type persistedState struct {
	Version     string                      `json:"version"`
	Checkpoints []model.KnowledgeCheckpoint `json:"checkpoints"`
}

func NewStore(path string) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("knowledge checkpoint store path is required")
	}
	cleanPath := filepath.Clean(path)
	store := &Store{
		path:        cleanPath,
		checkpoints: make(map[string]model.KnowledgeCheckpoint),
		order:       make([]string, 0),
	}
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
		return nil, err
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) SaveKnowledgeCheckpoint(_ context.Context, checkpoint model.KnowledgeCheckpoint) error {
	if err := checkpoint.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	checkpointID := checkpoint.CheckpointID
	if _, exists := s.checkpoints[checkpointID]; !exists {
		s.order = append(s.order, checkpointID)
	}
	s.checkpoints[checkpointID] = checkpoint
	return s.flush()
}

func (s *Store) FindKnowledgeCheckpoint(_ context.Context, checkpointID string) (model.KnowledgeCheckpoint, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	checkpoint, ok := s.checkpoints[checkpointID]
	return checkpoint, ok, nil
}

func (s *Store) ListKnowledgeCheckpoints(_ context.Context, filter query.KnowledgeCheckpointFilter) ([]model.KnowledgeCheckpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	prefix := strings.TrimSpace(filter.Prefix)
	items := make([]model.KnowledgeCheckpoint, 0, limit)
	for i := len(s.order) - 1; i >= 0 && len(items) < limit; i-- {
		checkpointID := s.order[i]
		if prefix != "" && !strings.HasPrefix(checkpointID, prefix) {
			continue
		}
		if checkpoint, ok := s.checkpoints[checkpointID]; ok {
			items = append(items, checkpoint)
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
	for _, checkpoint := range state.Checkpoints {
		if err := checkpoint.Validate(); err != nil {
			continue
		}
		if _, exists := s.checkpoints[checkpoint.CheckpointID]; !exists {
			s.order = append(s.order, checkpoint.CheckpointID)
		}
		s.checkpoints[checkpoint.CheckpointID] = checkpoint
	}
	return nil
}

func (s *Store) flush() error {
	state := persistedState{
		Version:     storeVersion,
		Checkpoints: make([]model.KnowledgeCheckpoint, 0, len(s.order)),
	}
	for _, checkpointID := range s.order {
		if checkpoint, ok := s.checkpoints[checkpointID]; ok {
			state.Checkpoints = append(state.Checkpoints, checkpoint)
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
