package service

import (
	"context"
	"errors"
	"strings"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

const (
	agentJobExternalLeaseTargetKind            = "agent_job_external_lease"
	agentJobExternalLeaseEnableAction          = "enable"
	defaultAgentJobExternalLeaseExecutionOwner = "python_ai_worker_with_nats_result_ack"
)

type agentJobExternalLeasePreflightPlanner interface {
	PlanAgentJobExternalLease(ctx context.Context, cmd command.PlanAgentJobExternalLeaseCommand) (query.AgentJobExternalLeasePlanView, error)
}

type agentJobExternalLeaseControlPreflightChecker interface {
	CheckControlMutationPreflight(ctx context.Context, preflight query.ControlMutationPreflight) (query.ControlMutationPreflightView, error)
}

type AgentJobExternalLeasePreflightService struct {
	planner   agentJobExternalLeasePreflightPlanner
	preflight agentJobExternalLeaseControlPreflightChecker
}

func NewAgentJobExternalLeasePreflightService(
	planner agentJobExternalLeasePreflightPlanner,
	preflight agentJobExternalLeaseControlPreflightChecker,
) *AgentJobExternalLeasePreflightService {
	return &AgentJobExternalLeasePreflightService{
		planner:   planner,
		preflight: preflight,
	}
}

func (s *AgentJobExternalLeasePreflightService) CheckAgentJobExternalLeasePreflight(
	ctx context.Context,
	filter query.AgentJobExternalLeasePreflightFilter,
) (query.AgentJobExternalLeasePreflightView, error) {
	if err := ctx.Err(); err != nil {
		return query.AgentJobExternalLeasePreflightView{}, err
	}
	if s == nil || s.planner == nil {
		return query.AgentJobExternalLeasePreflightView{}, errors.New("agent job external lease preflight requires planner")
	}

	desiredExecutionOwner := strings.TrimSpace(filter.DesiredExecutionOwner)
	if desiredExecutionOwner == "" {
		desiredExecutionOwner = defaultAgentJobExternalLeaseExecutionOwner
	}
	targetID := strings.TrimSpace(filter.TargetID)
	if targetID == "" {
		targetID = desiredExecutionOwner
	}
	operatorID := strings.TrimSpace(filter.OperatorID)
	approvalID := strings.TrimSpace(filter.ApprovalID)

	plan, err := s.planner.PlanAgentJobExternalLease(ctx, command.PlanAgentJobExternalLeaseCommand{
		Readiness: command.CheckAgentJobExternalLeaseReadinessCommand{
			JobLimit:          filter.JobLimit,
			EventLimit:        filter.EventLimit,
			StaleAfterSeconds: filter.StaleAfterSeconds,
		},
		DesiredExecutionOwner: desiredExecutionOwner,
	})
	if err != nil {
		return query.AgentJobExternalLeasePreflightView{}, err
	}

	base := query.AgentJobExternalLeasePreflightView{
		TargetKind:                agentJobExternalLeaseTargetKind,
		TargetID:                  targetID,
		Action:                    agentJobExternalLeaseEnableAction,
		DesiredExecutionOwner:     desiredExecutionOwner,
		CurrentExecutionOwner:     plan.CurrentExecutionOwner,
		RecommendedExecutionOwner: plan.RecommendedExecutionOwner,
		OperatorID:                operatorID,
		ApprovalID:                approvalID,
		Plan:                      plan,
		SideEffect:                "none",
		Notes: []string{
			"read-only agent_job external lease cutover preflight; no queue acknowledgement, runtime mutation, worker startup, or AI execution is performed",
			"operator approval and control mutation audit records must be created explicitly through their own endpoints",
		},
	}
	if !plan.Ready {
		base.Reason = "agent_job_external_lease_plan_not_ready"
		base.Blockers = append([]string(nil), plan.Blockers...)
		if len(base.Blockers) == 0 && strings.TrimSpace(plan.Decision) != "" {
			base.Blockers = []string{plan.Decision}
		}
		return base, nil
	}
	if s.preflight == nil {
		base.Reason = "control_mutation_preflight_unavailable"
		base.Blockers = []string{"control_mutation_preflight_unavailable"}
		return base, nil
	}

	control, err := s.preflight.CheckControlMutationPreflight(ctx, query.ControlMutationPreflight{
		TargetKind: agentJobExternalLeaseTargetKind,
		TargetID:   targetID,
		Action:     agentJobExternalLeaseEnableAction,
		OperatorID: operatorID,
		ApprovalID: approvalID,
	})
	if err != nil {
		return query.AgentJobExternalLeasePreflightView{}, err
	}
	base.ControlPreflight = control
	base.SuggestedAudit = control.SuggestedAudit
	if !control.Ready {
		base.Reason = control.Reason
		base.Blockers = append([]string(nil), control.Blockers...)
		if len(base.Blockers) == 0 && control.Reason != "" {
			base.Blockers = []string{control.Reason}
		}
		return base, nil
	}

	base.Ready = true
	base.Reason = "agent_job_external_lease_preflight_ready"
	return base, nil
}
