package observetargetstore

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

const storeVersion = "2026-05-31.observetargetstore.v1"

type Store struct {
	mu      sync.Mutex
	path    string
	targets map[string]model.ObserveTarget
}

type persistedState struct {
	Version string                `json:"version"`
	Targets []model.ObserveTarget `json:"targets"`
}

func NewStore(path string) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("observe target store path is required")
	}
	cleanPath := filepath.Clean(path)
	store := &Store{
		path:    cleanPath,
		targets: make(map[string]model.ObserveTarget),
	}
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
		return nil, err
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) SaveObserveTargets(_ context.Context, targets []model.ObserveTarget) error {
	if s == nil {
		return errors.New("observe target store is nil")
	}
	next := make(map[string]model.ObserveTarget, len(targets))
	for _, target := range targets {
		if err := target.Validate(); err != nil {
			return err
		}
		next[target.TargetID] = target
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.targets = next
	return s.flush()
}

func (s *Store) ListObserveTargets(_ context.Context) ([]model.ObserveTarget, error) {
	if s == nil {
		return nil, errors.New("observe target store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return sortedTargets(s.targets), nil
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
	for _, target := range state.Targets {
		if err := target.Validate(); err != nil {
			continue
		}
		s.targets[target.TargetID] = target
	}
	return nil
}

func (s *Store) flush() error {
	state := persistedState{
		Version: storeVersion,
		Targets: sortedTargets(s.targets),
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

func sortedTargets(targets map[string]model.ObserveTarget) []model.ObserveTarget {
	items := make([]model.ObserveTarget, 0, len(targets))
	for _, target := range targets {
		items = append(items, target)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Channel.Kind != items[j].Channel.Kind {
			return items[i].Channel.Kind < items[j].Channel.Kind
		}
		if items[i].Channel.AccountID != items[j].Channel.AccountID {
			return items[i].Channel.AccountID < items[j].Channel.AccountID
		}
		if items[i].Channel.ConversationType != items[j].Channel.ConversationType {
			return items[i].Channel.ConversationType < items[j].Channel.ConversationType
		}
		if items[i].Channel.ConversationID != items[j].Channel.ConversationID {
			return items[i].Channel.ConversationID < items[j].Channel.ConversationID
		}
		return items[i].TargetID < items[j].TargetID
	})
	return items
}
