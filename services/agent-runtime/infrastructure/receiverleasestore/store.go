package receiverleasestore

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

const storeVersion = "2026-05-31.receiverleasestore.v1"

type Store struct {
	mu     sync.Mutex
	path   string
	leases map[string]model.ReceiverLease
}

type persistedState struct {
	Version string                `json:"version"`
	Leases  []model.ReceiverLease `json:"leases"`
}

func NewStore(path string) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("receiver lease store path is required")
	}
	cleanPath := filepath.Clean(path)
	store := &Store{
		path:   cleanPath,
		leases: make(map[string]model.ReceiverLease),
	}
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
		return nil, err
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) SaveReceiverLease(_ context.Context, lease model.ReceiverLease) error {
	if s == nil {
		return errors.New("receiver lease store is nil")
	}
	if err := lease.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.leases[lease.ReceiverID] = lease
	return s.flush()
}

func (s *Store) DeleteReceiverLease(_ context.Context, receiverID string) error {
	if s == nil {
		return errors.New("receiver lease store is nil")
	}
	receiverID = strings.TrimSpace(receiverID)
	if receiverID == "" {
		return errors.New("receiver lease delete requires receiver_id")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.leases, receiverID)
	return s.flush()
}

func (s *Store) ListReceiverLeases(_ context.Context) ([]model.ReceiverLease, error) {
	if s == nil {
		return nil, errors.New("receiver lease store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return receiverLeaseItems(s.leases), nil
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
		s.leases[lease.ReceiverID] = lease
	}
	return nil
}

func (s *Store) flush() error {
	state := persistedState{
		Version: storeVersion,
		Leases:  receiverLeaseItems(s.leases),
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

func receiverLeaseItems(items map[string]model.ReceiverLease) []model.ReceiverLease {
	result := make([]model.ReceiverLease, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	return model.SortedReceiverLeases(result)
}
