package outboxstore

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

const storeVersion = "2026-05-30.outboxstore.v1"

// Store persists outbound delivery state for Go-owned outbox recovery.
type Store struct {
	mu         sync.Mutex
	path       string
	deliveries map[string]model.OutboxDelivery
	order      []string
	queue      []string
}

type persistedState struct {
	Version    string                 `json:"version"`
	Deliveries []model.OutboxDelivery `json:"deliveries"`
	Queue      []string               `json:"queue,omitempty"`
}

func NewStore(path string) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("outbox store path is required")
	}
	cleanPath := filepath.Clean(path)
	store := &Store{
		path:       cleanPath,
		deliveries: make(map[string]model.OutboxDelivery),
		order:      make([]string, 0),
		queue:      make([]string, 0),
	}
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
		return nil, err
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) SaveOutboxDelivery(_ context.Context, delivery model.OutboxDelivery) error {
	if err := delivery.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.deliveries[delivery.Message.EventID]; !exists {
		s.order = append(s.order, delivery.Message.EventID)
	}
	s.deliveries[delivery.Message.EventID] = delivery
	return s.flush()
}

func (s *Store) FindOutboxDelivery(_ context.Context, eventID string) (model.OutboxDelivery, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delivery, ok := s.deliveries[eventID]
	if !ok {
		return model.OutboxDelivery{}, false, nil
	}
	return delivery, true, nil
}

func (s *Store) ListOutboxDeliveries(_ context.Context, limit int) ([]model.OutboxDelivery, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 || limit > 200 {
		limit = 50
	}
	items := make([]model.OutboxDelivery, 0, limit)
	for i := len(s.order) - 1; i >= 0 && len(items) < limit; i-- {
		eventID := s.order[i]
		if delivery, ok := s.deliveries[eventID]; ok {
			items = append(items, delivery)
		}
	}
	return items, nil
}

func (s *Store) EnqueueOutboxDelivery(_ context.Context, delivery model.OutboxDelivery) error {
	if err := delivery.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	eventID := delivery.Message.EventID
	if _, exists := s.deliveries[eventID]; !exists {
		s.order = append(s.order, eventID)
	}
	s.deliveries[eventID] = delivery
	if !contains(s.queue, eventID) {
		s.queue = append(s.queue, eventID)
	}
	return s.flush()
}

func (s *Store) QueueIDs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]string, len(s.queue))
	copy(items, s.queue)
	return items
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
	for _, delivery := range state.Deliveries {
		eventID := delivery.Message.EventID
		if eventID == "" {
			continue
		}
		if err := delivery.Validate(); err != nil {
			continue
		}
		if _, exists := s.deliveries[eventID]; !exists {
			s.order = append(s.order, eventID)
		}
		s.deliveries[eventID] = delivery
	}
	for _, eventID := range state.Queue {
		eventID = strings.TrimSpace(eventID)
		if eventID == "" {
			continue
		}
		if _, ok := s.deliveries[eventID]; !ok {
			continue
		}
		if !contains(s.queue, eventID) {
			s.queue = append(s.queue, eventID)
		}
	}
	return nil
}

func (s *Store) flush() error {
	state := persistedState{
		Version:    storeVersion,
		Deliveries: make([]model.OutboxDelivery, 0, len(s.order)),
		Queue:      append(make([]string, 0, len(s.queue)), s.queue...),
	}
	for _, eventID := range s.order {
		if delivery, ok := s.deliveries[eventID]; ok {
			state.Deliveries = append(state.Deliveries, delivery)
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

func contains(items []string, needle string) bool {
	for _, item := range items {
		if item == needle {
			return true
		}
	}
	return false
}
