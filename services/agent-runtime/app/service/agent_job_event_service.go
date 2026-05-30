package service

import (
	"context"
	"errors"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/assembler"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type AgentJobEventService struct {
	store outport.AgentJobEventStore
}

func NewAgentJobEventService(store outport.AgentJobEventStore) *AgentJobEventService {
	return &AgentJobEventService{store: store}
}

func (s *AgentJobEventService) List(ctx context.Context, filter query.AgentJobEventFilter) ([]query.AgentJobEventView, error) {
	if s == nil || s.store == nil {
		return nil, errors.New("agent job event service requires store")
	}
	items, err := s.store.ListAgentJobEvents(ctx, filter)
	if err != nil {
		return nil, err
	}
	return assembler.ToAgentJobEventViews(items), nil
}
