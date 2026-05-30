package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/assembler"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	domainservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/service"
)

type DeliveryDispatchService struct {
	repository outport.OutboxRepository
	planner    domainservice.DeliveryPlanner
	adapters   []outport.DeliveryAdapter
}

func NewDeliveryDispatchService(repository outport.OutboxRepository) *DeliveryDispatchService {
	return NewDeliveryDispatchServiceWithAdapters(repository)
}

func NewDeliveryDispatchServiceWithAdapters(
	repository outport.OutboxRepository,
	adapters ...outport.DeliveryAdapter,
) *DeliveryDispatchService {
	return &DeliveryDispatchService{
		repository: repository,
		planner:    domainservice.NewDeliveryPlanner(),
		adapters:   adapters,
	}
}

func (s *DeliveryDispatchService) Plan(ctx context.Context, cmd command.PlanDeliveryDispatchCommand) (query.DeliveryDispatchPlanView, error) {
	plan, err := s.planModel(ctx, cmd.EventID, cmd.ChannelByAccount)
	if err != nil {
		return query.DeliveryDispatchPlanView{}, err
	}
	return assembler.ToDeliveryDispatchPlanView(plan), nil
}

func (s *DeliveryDispatchService) Readiness(ctx context.Context, cmd command.CheckDeliveryDispatchReadinessCommand) (query.DeliveryDispatchReadinessView, error) {
	plan, err := s.planModel(ctx, cmd.EventID, cmd.ChannelByAccount)
	if err != nil {
		return query.DeliveryDispatchReadinessView{}, err
	}
	missingChannels := make([]string, 0)
	seen := make(map[string]struct{})
	for _, step := range plan.Steps {
		channel := strings.TrimSpace(step.Channel)
		if channel == "" {
			continue
		}
		if _, ok := seen[channel]; ok {
			continue
		}
		seen[channel] = struct{}{}
		if _, ok := s.adapterFor(channel); !ok {
			missingChannels = append(missingChannels, channel)
		}
	}
	reason := "delivery_adapter_ready"
	ready := len(missingChannels) == 0
	if !ready {
		reason = "delivery_adapter_unavailable"
	}
	return query.DeliveryDispatchReadinessView{
		EventID:         plan.EventID,
		Channel:         plan.Channel,
		Ready:           ready,
		Reason:          reason,
		MissingChannels: missingChannels,
		Plan:            assembler.ToDeliveryDispatchPlanView(plan),
		Attributes: map[string]string{
			"planned_by":  plan.Attributes["planned_by"],
			"checked_by":  "agent_runtime_delivery_readiness",
			"side_effect": "none",
		},
	}, nil
}

func (s *DeliveryDispatchService) Dispatch(ctx context.Context, cmd command.DispatchDeliveryCommand) (query.DeliveryDispatchResultView, error) {
	plan, err := s.planModel(ctx, cmd.EventID, cmd.ChannelByAccount)
	if err != nil {
		return query.DeliveryDispatchResultView{}, err
	}
	results := make([]model.DeliveryDispatchResult, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		adapter, ok := s.adapterFor(step.Channel)
		if !ok {
			return query.DeliveryDispatchResultView{}, DeliveryDispatchError{
				Kind:    model.DeliveryErrorSenderUnavailable,
				Message: fmt.Sprintf("delivery adapter unavailable for channel %q", step.Channel),
			}
		}
		result, err := adapter.DispatchDeliveryStep(ctx, step)
		if err != nil {
			return query.DeliveryDispatchResultView{}, normalizeDeliveryDispatchError(err)
		}
		results = append(results, result)
	}
	return assembler.ToDeliveryDispatchResultView(model.DeliveryDispatchExecution{
		EventID:   plan.EventID,
		StepCount: len(results),
		Results:   results,
		Attributes: map[string]string{
			"planned_by":    plan.Attributes["planned_by"],
			"dispatched_by": "agent_runtime_delivery_dispatcher",
		},
	}), nil
}

func (s *DeliveryDispatchService) planModel(
	ctx context.Context,
	eventID string,
	channelByAccount map[string]string,
) (model.DeliveryDispatchPlan, error) {
	if s == nil || s.repository == nil {
		return model.DeliveryDispatchPlan{}, errors.New("delivery dispatch service requires repository")
	}
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return model.DeliveryDispatchPlan{}, errors.New("outbox event id required")
	}
	delivery, ok, err := s.repository.FindOutboxDelivery(ctx, eventID)
	if err != nil {
		return model.DeliveryDispatchPlan{}, err
	}
	if !ok {
		return model.DeliveryDispatchPlan{}, errors.New("outbox delivery not found")
	}
	plan, err := s.planner.Plan(delivery, domainservice.DeliveryPlannerConfig{
		ChannelByAccount: channelByAccount,
	})
	if err != nil {
		return model.DeliveryDispatchPlan{}, err
	}
	return plan, nil
}

func (s *DeliveryDispatchService) adapterFor(channel string) (outport.DeliveryAdapter, bool) {
	channel = strings.TrimSpace(channel)
	if channel == "" {
		return nil, false
	}
	for _, adapter := range s.adapters {
		if adapter != nil && adapter.SupportsDeliveryChannel(channel) {
			return adapter, true
		}
	}
	return nil, false
}

type DeliveryDispatchError struct {
	Kind    model.DeliveryErrorKind
	Message string
}

func (e DeliveryDispatchError) Error() string {
	return e.Message
}

func (e DeliveryDispatchError) DeliveryErrorKind() string {
	return string(model.NormalizeDeliveryErrorKind(string(e.Kind)))
}

func normalizeDeliveryDispatchError(err error) DeliveryDispatchError {
	if err == nil {
		return DeliveryDispatchError{Kind: model.DeliveryErrorUnknown, Message: "delivery dispatch failed"}
	}
	kind := model.DeliveryErrorUnknown
	if kinded, ok := err.(interface{ DeliveryErrorKind() string }); ok {
		kind = model.NormalizeDeliveryErrorKind(kinded.DeliveryErrorKind())
	}
	return DeliveryDispatchError{
		Kind:    kind,
		Message: err.Error(),
	}
}
