package inboxstore

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

const storeVersion = "2026-05-30.inboxstore.v1"

// Store persists raw inbound/observed message events for replay and group memory.
type Store struct {
	mu     sync.Mutex
	path   string
	events map[string]model.InboxEvent
	order  []string
}

type persistedState struct {
	Version string             `json:"version"`
	Events  []model.InboxEvent `json:"events"`
}

func NewStore(path string) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("inbox store path is required")
	}
	cleanPath := filepath.Clean(path)
	store := &Store{
		path:   cleanPath,
		events: make(map[string]model.InboxEvent),
		order:  make([]string, 0),
	}
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
		return nil, err
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) SaveInboxEvent(_ context.Context, event model.InboxEvent) error {
	if err := event.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	eventID := event.EventID()
	if _, exists := s.events[eventID]; exists {
		return nil
	}
	s.order = append(s.order, eventID)
	s.events[eventID] = event
	return s.flush()
}

func (s *Store) FindInboxEvent(_ context.Context, eventID string) (model.InboxEvent, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	event, ok := s.events[eventID]
	if !ok {
		return model.InboxEvent{}, false, nil
	}
	return event, true, nil
}

func (s *Store) ListInboxEvents(_ context.Context, filter query.InboxEventFilter) ([]model.InboxEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	items := make([]model.InboxEvent, 0, limit)
	for i := len(s.order) - 1; i >= 0 && len(items) < limit; i-- {
		eventID := s.order[i]
		if event, ok := s.events[eventID]; ok && matchesInboxEventFilter(event, filter) {
			items = append(items, event)
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
	for _, event := range state.Events {
		eventID := event.EventID()
		if eventID == "" {
			continue
		}
		if err := event.Validate(); err != nil {
			continue
		}
		if _, exists := s.events[eventID]; !exists {
			s.order = append(s.order, eventID)
		}
		s.events[eventID] = event
	}
	return nil
}

func (s *Store) flush() error {
	state := persistedState{
		Version: storeVersion,
		Events:  make([]model.InboxEvent, 0, len(s.order)),
	}
	for _, eventID := range s.order {
		if event, ok := s.events[eventID]; ok {
			state.Events = append(state.Events, event)
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

func matchesInboxEventFilter(event model.InboxEvent, filter query.InboxEventFilter) bool {
	envelope := event.Envelope
	if filter.ChannelKind != "" && string(envelope.Channel.Kind) != filter.ChannelKind {
		return false
	}
	if filter.AccountID != "" && envelope.Channel.AccountID != filter.AccountID {
		return false
	}
	if filter.ConversationID != "" && envelope.Channel.ConversationID != filter.ConversationID {
		return false
	}
	if filter.ConversationType != "" && string(envelope.Channel.ConversationType) != filter.ConversationType {
		return false
	}
	if filter.SenderID != "" && envelope.Sender.ID != filter.SenderID {
		return false
	}
	if filter.DecisionAction != "" && string(event.Decision.Action) != filter.DecisionAction {
		return false
	}
	if filter.ObserveOnly != "" && parseBoolFilter(filter.ObserveOnly) != event.ObserveOnly() {
		return false
	}
	return true
}

func parseBoolFilter(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "true" || value == "1" || value == "yes"
}
