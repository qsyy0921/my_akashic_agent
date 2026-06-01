package operatorapprovalstore

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

const storeVersion = "2026-06-01.operatorapprovalstore.v1"

type Store struct {
	mu        sync.Mutex
	path      string
	approvals map[string]model.OperatorApproval
}

type persistedState struct {
	Version   string                   `json:"version"`
	Approvals []model.OperatorApproval `json:"approvals"`
}

func NewStore(path string) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("operator approval store path is required")
	}
	cleanPath := filepath.Clean(path)
	store := &Store{
		path:      cleanPath,
		approvals: make(map[string]model.OperatorApproval),
	}
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
		return nil, err
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) SaveOperatorApproval(_ context.Context, approval model.OperatorApproval) error {
	if s == nil {
		return errors.New("operator approval store is nil")
	}
	if err := approval.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.approvals[approval.ApprovalID] = approval
	return s.flush()
}

func (s *Store) ListOperatorApprovals(_ context.Context) ([]model.OperatorApproval, error) {
	if s == nil {
		return nil, errors.New("operator approval store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return model.SortedOperatorApprovals(operatorApprovalItems(s.approvals)), nil
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
	for _, approval := range state.Approvals {
		if err := approval.Validate(); err != nil {
			continue
		}
		s.approvals[approval.ApprovalID] = approval
	}
	return nil
}

func (s *Store) flush() error {
	state := persistedState{
		Version:   storeVersion,
		Approvals: model.SortedOperatorApprovals(operatorApprovalItems(s.approvals)),
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

func operatorApprovalItems(items map[string]model.OperatorApproval) []model.OperatorApproval {
	result := make([]model.OperatorApproval, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	return result
}
