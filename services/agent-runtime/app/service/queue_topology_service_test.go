package service

import (
	"context"
	"testing"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

func TestQueueTopologyShowsExternalLeaseBoundaries(t *testing.T) {
	backend := NewQueueBackendService(queueTopologyBackendForTest())
	service := NewQueueTopologyService(backend)

	view, err := service.GetQueueTopology(context.Background())
	if err != nil {
		t.Fatalf("queue topology: %v", err)
	}

	if view.Provider != "nats_jetstream" || view.Mode != "external_lease" {
		t.Fatalf("unexpected provider/mode: %#v", view)
	}
	if view.SideEffect != "none" {
		t.Fatalf("topology must be read-only: %#v", view)
	}
	if !view.ExternalLeaseReady || view.ExecutionScope != "outbox_delivery_only" {
		t.Fatalf("unexpected external lease state: %#v", view)
	}
	outbox := findQueueTopologyWorkKind(t, view.WorkKinds, "outbox_delivery")
	if !outbox.Allowed || outbox.ExecutionOwner != "nats_external_lease" || outbox.AckOwner != "nats_external_lease" {
		t.Fatalf("unexpected outbox topology: %#v", outbox)
	}
	agentJob := findQueueTopologyWorkKind(t, view.WorkKinds, "agent_job")
	if agentJob.Allowed || agentJob.ExecutionOwner != "python_ai_worker_state_store_lease" || agentJob.AckOwner != "go_state_store_api" {
		t.Fatalf("unexpected agent job topology: %#v", agentJob)
	}
	if len(agentJob.Blockers) != 1 || agentJob.Blockers[0] != "agent_job_result_ack_disabled:set result ack flags" {
		t.Fatalf("unexpected agent job blockers: %#v", agentJob.Blockers)
	}
	if len(view.Edges) != 2 {
		t.Fatalf("expected one edge per work kind: %#v", view.Edges)
	}
	if len(view.Blockers) == 0 {
		t.Fatalf("expected topology blockers to surface blocked work kinds")
	}
}

func queueTopologyBackendForTest() query.QueueBackendView {
	return query.QueueBackendView{
		Provider:                "nats_jetstream",
		Mode:                    "external_lease",
		MigrationPhase:          "first_external_mq",
		ExternalQueueConfigured: true,
		ExternalQueueActive:     true,
		StateStoreAuthoritative: true,
		LeaseOwner:              "go_state_store_api",
		ConsumerModel:           "durable_pull",
		ConsumerConcurrency:     8,
		MaxInFlight:             64,
		OutboxQueueSource:       "outbox_state_store",
		AgentJobQueueSource:     "agent_job_state_store",
		OutboxExecutionOwner:    "nats_external_lease",
		AgentJobExecutionOwner:  "python_ai_worker_state_store_lease",
		RecommendedFirstBackend: "nats_jetstream",
		SelectedProviderCapability: &query.QueueProviderCapabilityView{
			Provider:                    "nats_jetstream",
			Status:                      "selected",
			Recommended:                 true,
			Implemented:                 true,
			SupportsConcurrentConsumers: true,
			SupportsDelayedNack:         true,
		},
		ExternalLease: &query.QueueExternalLeaseGate{
			Enabled:          true,
			CutoverRequested: true,
			AllowExecution:   true,
			GateState:        "ready",
			ExecutionScope:   "outbox_delivery_only",
			AckPolicy:        "ack_after_success",
			AllowedWorkKinds: []string{"outbox_delivery"},
			BlockedWorkKinds: []query.QueueExternalLeaseBlock{{
				WorkKind:       "agent_job",
				Reason:         "agent_job_result_ack_disabled",
				RequiredChange: "set result ack flags",
			}},
		},
	}
}

func findQueueTopologyWorkKind(t *testing.T, items []query.QueueTopologyWorkKind, workKind string) query.QueueTopologyWorkKind {
	t.Helper()
	for _, item := range items {
		if item.WorkKind == workKind {
			return item
		}
	}
	t.Fatalf("missing work kind %s in %#v", workKind, items)
	return query.QueueTopologyWorkKind{}
}
