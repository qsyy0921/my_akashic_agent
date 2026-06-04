package service_test

import (
	"context"
	"testing"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
)

func TestAgentJobExternalLeasePreflightServiceBlocksWhenPlanNotReady(t *testing.T) {
	service := appservice.NewAgentJobExternalLeasePreflightService(
		staticAgentJobExternalLeasePreflightPlanner{view: query.AgentJobExternalLeasePlanView{
			Ready:                     false,
			Decision:                  "blocked",
			DesiredExecutionOwner:     "python_ai_worker_with_nats_result_ack",
			RecommendedExecutionOwner: "python_ai_worker_with_nats_result_ack",
			CurrentExecutionOwner:     "python_ai_worker_state_store_lease",
			Blockers:                  []string{"agent_job_external_lease_readiness_not_ready"},
		}},
		nil,
	)

	view, err := service.CheckAgentJobExternalLeasePreflight(context.Background(), query.AgentJobExternalLeasePreflightFilter{
		DesiredExecutionOwner: "python_ai_worker_with_nats_result_ack",
		OperatorID:            "qsyy",
	})
	if err != nil {
		t.Fatalf("check agent job external lease preflight: %v", err)
	}
	if view.Ready ||
		view.Reason != "agent_job_external_lease_plan_not_ready" ||
		len(view.Blockers) != 1 ||
		view.Blockers[0] != "agent_job_external_lease_readiness_not_ready" {
		t.Fatalf("unexpected blocked preflight view: %+v", view)
	}
}

func TestAgentJobExternalLeasePreflightServiceRequiresApprovalAfterPlanReady(t *testing.T) {
	service := appservice.NewAgentJobExternalLeasePreflightService(
		staticAgentJobExternalLeasePreflightPlanner{view: query.AgentJobExternalLeasePlanView{
			Ready:                     true,
			Decision:                  "ready",
			DesiredExecutionOwner:     "python_ai_worker_with_nats_result_ack",
			RecommendedExecutionOwner: "python_ai_worker_with_nats_result_ack",
			CurrentExecutionOwner:     "python_ai_worker_state_store_lease",
		}},
		staticAgentJobExternalLeaseControlPreflight{view: query.ControlMutationPreflightView{
			Ready:      false,
			Reason:     "missing_approval_id",
			Blockers:   []string{"missing_approval_id"},
			TargetKind: "agent_job_external_lease",
			TargetID:   "python_ai_worker_with_nats_result_ack",
			Action:     "enable",
			OperatorID: "qsyy",
			ApprovalID: "",
			SideEffect: "none",
		}},
	)

	view, err := service.CheckAgentJobExternalLeasePreflight(context.Background(), query.AgentJobExternalLeasePreflightFilter{
		DesiredExecutionOwner: "python_ai_worker_with_nats_result_ack",
		OperatorID:            "qsyy",
	})
	if err != nil {
		t.Fatalf("check agent job external lease preflight: %v", err)
	}
	if view.Ready ||
		view.Reason != "missing_approval_id" ||
		len(view.Blockers) != 1 ||
		view.Blockers[0] != "missing_approval_id" {
		t.Fatalf("unexpected missing approval preflight view: %+v", view)
	}
}

func TestAgentJobExternalLeasePreflightServiceAllowsApprovedReadyPlan(t *testing.T) {
	service := appservice.NewAgentJobExternalLeasePreflightService(
		staticAgentJobExternalLeasePreflightPlanner{view: query.AgentJobExternalLeasePlanView{
			Ready:                     true,
			Decision:                  "ready",
			DesiredExecutionOwner:     "python_ai_worker_with_nats_result_ack",
			RecommendedExecutionOwner: "python_ai_worker_with_nats_result_ack",
			CurrentExecutionOwner:     "python_ai_worker_state_store_lease",
		}},
		staticAgentJobExternalLeaseControlPreflight{view: query.ControlMutationPreflightView{
			Ready:      true,
			Reason:     "approval_active",
			TargetKind: "agent_job_external_lease",
			TargetID:   "python_ai_worker_with_nats_result_ack",
			Action:     "enable",
			OperatorID: "qsyy",
			ApprovalID: "approval-a",
			SuggestedAudit: &query.ControlMutationSuggestedAuditView{
				TargetKind: "agent_job_external_lease",
				TargetID:   "python_ai_worker_with_nats_result_ack",
				Action:     "enable",
				Status:     "planned",
				OperatorID: "qsyy",
				ApprovalID: "approval-a",
			},
			SideEffect: "none",
		}},
	)

	view, err := service.CheckAgentJobExternalLeasePreflight(context.Background(), query.AgentJobExternalLeasePreflightFilter{
		DesiredExecutionOwner: "python_ai_worker_with_nats_result_ack",
		OperatorID:            "qsyy",
		ApprovalID:            "approval-a",
	})
	if err != nil {
		t.Fatalf("check agent job external lease preflight: %v", err)
	}
	if !view.Ready ||
		view.Reason != "agent_job_external_lease_preflight_ready" ||
		view.SuggestedAudit == nil ||
		view.SuggestedAudit.TargetKind != "agent_job_external_lease" ||
		view.SuggestedAudit.Action != "enable" ||
		view.SideEffect != "none" {
		t.Fatalf("unexpected ready preflight view: %+v", view)
	}
}

type staticAgentJobExternalLeasePreflightPlanner struct {
	view query.AgentJobExternalLeasePlanView
}

func (s staticAgentJobExternalLeasePreflightPlanner) PlanAgentJobExternalLease(context.Context, command.PlanAgentJobExternalLeaseCommand) (query.AgentJobExternalLeasePlanView, error) {
	return s.view, nil
}

type staticAgentJobExternalLeaseControlPreflight struct {
	view query.ControlMutationPreflightView
}

func (s staticAgentJobExternalLeaseControlPreflight) CheckControlMutationPreflight(context.Context, query.ControlMutationPreflight) (query.ControlMutationPreflightView, error) {
	return s.view, nil
}
