package service

import (
	"context"
	"testing"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

func TestAgentJobExternalLeasePlanReportsReadyToEnableResultAck(t *testing.T) {
	service := NewAgentJobExternalLeasePlanService(AgentJobExternalLeasePlanDeps{
		Readiness: staticAgentJobExternalLeasePlanReadiness{view: query.AgentJobExternalLeaseReadinessView{
			Ready:                   true,
			Reason:                  "agent_job_external_lease_ready",
			ExternalLeaseReady:      true,
			AgentJobResultAckReady:  true,
			StrictLeaseTokenEnabled: true,
			AgentJobWorkerReady:     true,
			ExecutionOwner:          "python_ai_worker_state_store_lease",
			QueueProvider:           "nats_jetstream",
			QueueMode:               "external_lease",
			ExecutionScope:          "outbox_delivery_and_agent_job_result_ack",
			AllowedWorkKinds:        []string{"outbox_delivery", "agent_job"},
			SideEffect:              "none",
		}},
	})

	view, err := service.PlanAgentJobExternalLease(context.Background(), command.PlanAgentJobExternalLeaseCommand{})
	if err != nil {
		t.Fatalf("plan agent job external lease: %v", err)
	}
	if view.Ready {
		t.Fatalf("plan should not report final ready until current owner matches desired: %+v", view)
	}
	if view.Decision != "ready_to_enable_result_ack" {
		t.Fatalf("unexpected decision: %+v", view)
	}
	if view.DesiredExecutionOwner != "python_ai_worker_with_nats_result_ack" ||
		view.CurrentExecutionOwner != "python_ai_worker_state_store_lease" ||
		view.RecommendedExecutionOwner != "python_ai_worker_with_nats_result_ack" {
		t.Fatalf("unexpected owners: %+v", view)
	}
	if len(view.Blockers) != 0 {
		t.Fatalf("unexpected blockers: %+v", view.Blockers)
	}
	if len(view.EnableSteps) == 0 ||
		view.EnableSteps[0].Env["AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED"] != "true" ||
		view.EnableSteps[0].Env["AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN"] != "true" {
		t.Fatalf("unexpected enable steps: %+v", view.EnableSteps)
	}
	if len(view.RollbackSteps) == 0 ||
		view.RollbackSteps[0].Env["AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED"] != "false" {
		t.Fatalf("unexpected rollback steps: %+v", view.RollbackSteps)
	}
	if view.SideEffect != "none" {
		t.Fatalf("plan must be read-only: %+v", view)
	}
}

func TestAgentJobExternalLeasePlanBlocksUntilReadinessPasses(t *testing.T) {
	service := NewAgentJobExternalLeasePlanService(AgentJobExternalLeasePlanDeps{
		Readiness: staticAgentJobExternalLeasePlanReadiness{view: query.AgentJobExternalLeaseReadinessView{
			Ready:               false,
			Reason:              "agent_job_external_lease_not_ready",
			ExternalLeaseReady:  true,
			AgentJobWorkerReady: false,
			ExecutionOwner:      "python_ai_worker_state_store_lease",
			Blockers: []string{
				"agent_job_worker_coverage_blocked",
				"queue_external_lease_agent_job_flag_disabled",
			},
			SideEffect: "none",
		}},
	})

	view, err := service.PlanAgentJobExternalLease(context.Background(), command.PlanAgentJobExternalLeaseCommand{
		DesiredExecutionOwner: "nats_result_ack",
	})
	if err != nil {
		t.Fatalf("plan agent job external lease: %v", err)
	}
	if view.Decision != "blocked" || len(view.Blockers) != 3 {
		t.Fatalf("unexpected blocked plan: %+v", view)
	}
	if !containsString(view.Blockers, "agent_job_external_lease_readiness_not_ready") ||
		!containsString(view.Blockers, "agent_job_worker_coverage_blocked") ||
		!containsString(view.Blockers, "queue_external_lease_agent_job_flag_disabled") {
		t.Fatalf("missing readiness blockers: %+v", view.Blockers)
	}
}

func TestAgentJobExternalLeasePlanSupportsStateStoreRollbackOwner(t *testing.T) {
	service := NewAgentJobExternalLeasePlanService(AgentJobExternalLeasePlanDeps{
		Readiness: staticAgentJobExternalLeasePlanReadiness{view: query.AgentJobExternalLeaseReadinessView{
			Ready:          true,
			ExecutionOwner: "python_ai_worker_with_nats_result_ack",
			SideEffect:     "none",
		}},
	})

	view, err := service.PlanAgentJobExternalLease(context.Background(), command.PlanAgentJobExternalLeaseCommand{
		DesiredExecutionOwner: "state_store",
	})
	if err != nil {
		t.Fatalf("plan rollback: %v", err)
	}
	if view.Decision != "ready_to_rollback_to_state_store" ||
		view.DesiredExecutionOwner != "python_ai_worker_state_store_lease" {
		t.Fatalf("unexpected rollback plan: %+v", view)
	}
	if len(view.EnableSteps) == 0 ||
		view.EnableSteps[0].Env["AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED"] != "false" {
		t.Fatalf("unexpected rollback enable steps: %+v", view.EnableSteps)
	}
}

type staticAgentJobExternalLeasePlanReadiness struct {
	view query.AgentJobExternalLeaseReadinessView
}

func (s staticAgentJobExternalLeasePlanReadiness) CheckAgentJobExternalLeaseReadiness(context.Context, command.CheckAgentJobExternalLeaseReadinessCommand) (query.AgentJobExternalLeaseReadinessView, error) {
	return s.view, nil
}
