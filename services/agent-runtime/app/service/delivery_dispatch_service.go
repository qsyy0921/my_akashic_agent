package service

import (
	"context"
	"errors"
	"strings"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/assembler"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	domainservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/service"
)

type DeliveryDispatchService struct {
	repository outport.OutboxRepository
	planner    domainservice.DeliveryPlanner
}

func NewDeliveryDispatchService(repository outport.OutboxRepository) *DeliveryDispatchService {
	return &DeliveryDispatchService{
		repository: repository,
		planner:    domainservice.NewDeliveryPlanner(),
	}
}

func (s *DeliveryDispatchService) Plan(ctx context.Context, cmd command.PlanDeliveryDispatchCommand) (query.DeliveryDispatchPlanView, error) {
	if s == nil || s.repository == nil {
		return query.DeliveryDispatchPlanView{}, errors.New("delivery dispatch service requires repository")
	}
	eventID := strings.TrimSpace(cmd.EventID)
	if eventID == "" {
		return query.DeliveryDispatchPlanView{}, errors.New("outbox event id required")
	}
	delivery, ok, err := s.repository.FindOutboxDelivery(ctx, eventID)
	if err != nil {
		return query.DeliveryDispatchPlanView{}, err
	}
	if !ok {
		return query.DeliveryDispatchPlanView{}, errors.New("outbox delivery not found")
	}
	plan, err := s.planner.Plan(delivery, domainservice.DeliveryPlannerConfig{
		ChannelByAccount: cmd.ChannelByAccount,
	})
	if err != nil {
		return query.DeliveryDispatchPlanView{}, err
	}
	return assembler.ToDeliveryDispatchPlanView(plan), nil
}
