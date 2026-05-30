package agentjobeventstore

import (
	"bufio"
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

// Store appends AgentJob lifecycle events to a JSONL stream.
type Store struct {
	mu   sync.Mutex
	path string
}

func NewStore(path string) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("agent job event store path is required")
	}
	cleanPath := filepath.Clean(path)
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
		return nil, err
	}
	return &Store{path: cleanPath}, nil
}

func (s *Store) AppendAgentJobEvent(_ context.Context, event model.AgentJobEvent) error {
	if err := event.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	raw, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(raw, '\n')); err != nil {
		return err
	}
	return nil
}

func (s *Store) ListAgentJobEvents(_ context.Context, filter query.AgentJobEventFilter) ([]model.AgentJobEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	file, err := os.Open(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return []model.AgentJobEvent{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	all := make([]model.AgentJobEvent, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event model.AgentJobEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		if err := event.Validate(); err != nil {
			continue
		}
		all = append(all, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	items := make([]model.AgentJobEvent, 0, limit)
	for i := len(all) - 1; i >= 0 && len(items) < limit; i-- {
		event := all[i]
		if filter.JobID != "" && event.JobID != filter.JobID {
			continue
		}
		if filter.JobType != "" && string(event.JobType) != filter.JobType {
			continue
		}
		if filter.EventType != "" && string(event.EventType) != filter.EventType {
			continue
		}
		items = append(items, event)
	}
	return items, nil
}
