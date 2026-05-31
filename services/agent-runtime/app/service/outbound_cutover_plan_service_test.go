package service

import (
	"context"
	"testing"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

func TestOutboundCutoverPlanReportsReadyLocalWorkerPlan(t *testing.T) {
	service := NewOutboundCutoverPlanService(OutboundCutoverPlanDeps{
		Readiness: staticOutboundCutoverReadiness{view: query.OutboundCutoverReadinessView{
			Ready:                  true,
			Reason:                 "outbound_cutover_ready",
			OneBotReady:            true,
			SmokeReady:             true,
			ExecutionReady:         true,
			ExecutionOwner:         "go_local_outbox_worker",
			LocalOutboxWorkerReady: true,
			ExpectedOneBotChannels: []string{"qq_1049511700", "qq_2365524513"},
			SideEffect:             "none",
		}},
		QueueBackend: staticRuntimeQueueBackend{view: query.QueueBackendView{
			Provider:             "local",
			Mode:                 "local_state_store",
			OutboxExecutionOwner: "go_local_outbox_worker",
		}},
	})

	view, err := service.PlanOutboundCutover(context.Background(), command.PlanOutboundCutoverCommand{})
	if err != nil {
		t.Fatalf("plan outbound cutover: %v", err)
	}
	if !view.Ready || view.Decision != "ready_to_cutover" {
		t.Fatalf("expected ready local cutover plan: %#v", view)
	}
	if view.DesiredExecutionOwner != "go_local_outbox_worker" || view.RecommendedExecutionOwner != "go_local_outbox_worker" {
		t.Fatalf("unexpected owners: %#v", view)
	}
	if view.SideEffect != "none" || len(view.Blockers) != 0 {
		t.Fatalf("unexpected side effect/blockers: %#v", view)
	}
	if len(view.EnableSteps) == 0 || view.EnableSteps[0].Env["AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED"] != "true" {
		t.Fatalf("expected local worker enable step: %#v", view.EnableSteps)
	}
	if got := view.EnableSteps[0].Env["AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT"]; got != "1049511700=qq_1049511700,2365524513=qq_2365524513" {
		t.Fatalf("unexpected channel mapping hint: %s", got)
	}
}

func TestOutboundCutoverPlanBlocksExternalLeaseUntilGateReady(t *testing.T) {
	service := NewOutboundCutoverPlanService(OutboundCutoverPlanDeps{
		Readiness: staticOutboundCutoverReadiness{view: query.OutboundCutoverReadinessView{
			Ready:                    false,
			Reason:                   "outbound_cutover_not_ready",
			OneBotReady:              true,
			SmokeReady:               true,
			ExecutionOwner:           "go_state_store_api",
			ExternalLeaseOutboxReady: false,
			Blockers:                 []string{"outbox_execution_path_not_ready"},
			SideEffect:               "none",
		}},
		QueueBackend: staticRuntimeQueueBackend{view: query.QueueBackendView{
			Provider:             "nats_jetstream",
			Mode:                 "external_lease",
			OutboxExecutionOwner: "go_state_store_api",
			DSNConfigured:        true,
			ExternalLease: &query.QueueExternalLeaseGate{
				Enabled:  true,
				Blockers: []string{"dual_read_smoke_passed", "state_lease_workers_disabled"},
			},
		}},
	})

	view, err := service.PlanOutboundCutover(context.Background(), command.PlanOutboundCutoverCommand{
		DesiredExecutionOwner: "nats_external_lease",
	})
	if err != nil {
		t.Fatalf("plan outbound cutover: %v", err)
	}
	if view.Ready || view.Decision != "blocked" {
		t.Fatalf("expected blocked external lease plan: %#v", view)
	}
	if view.DesiredExecutionOwner != "nats_external_lease" || view.RecommendedExecutionOwner != "nats_external_lease" {
		t.Fatalf("unexpected external owners: %#v", view)
	}
	if !containsString(view.Blockers, "external_lease_outbox_not_ready") || !containsString(view.Blockers, "dual_read_smoke_passed") {
		t.Fatalf("expected external lease blockers: %#v", view.Blockers)
	}
	if len(view.RollbackSteps) == 0 || view.RollbackSteps[0].Env["AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER"] != "false" {
		t.Fatalf("expected external lease rollback step: %#v", view.RollbackSteps)
	}
}

type staticOutboundCutoverReadiness struct {
	view query.OutboundCutoverReadinessView
}

func (s staticOutboundCutoverReadiness) CheckOutboundCutoverReadiness(context.Context, command.CheckOutboundCutoverReadinessCommand) (query.OutboundCutoverReadinessView, error) {
	return s.view, nil
}
