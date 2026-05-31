package service

import (
	"context"
	"testing"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

func TestAgentJobExternalLeaseReadinessReportsReady(t *testing.T) {
	service := NewAgentJobExternalLeaseReadinessService(AgentJobExternalLeaseReadinessDeps{
		QueueBackend: staticRuntimeQueueBackend{view: query.QueueBackendView{
			Provider:               "nats_jetstream",
			Mode:                   "external_lease",
			AgentJobExecutionOwner: "python_ai_worker_with_nats_result_ack",
			ExternalLease: &query.QueueExternalLeaseGate{
				AllowExecution:   true,
				ExecutionScope:   "outbox_delivery_and_agent_job_result_ack",
				AllowedWorkKinds: []string{"outbox_delivery", "agent_job"},
			},
		}},
		RuntimeConfig: staticRuntimeConfig{view: query.RuntimeConfigView{
			Workers: query.RuntimeWorkerConfigView{
				AgentJobStrictLeaseToken:         true,
				QueueExternalLeaseAgentJobEnable: true,
			},
			SideEffect: "none",
		}},
		AgentJobs: staticRuntimeAgentJobMetrics{view: query.AgentJobMetricsView{
			Pressure: query.AgentJobPressureMetricsView{
				ByType: []query.AgentJobTypePressureView{{
					JobType:      "group_memory_extract",
					Pending:      1,
					HighPressure: false,
				}},
			},
		}},
		AgentWorkers: staticRuntimeAgentWorkers{view: query.AgentWorkerStatusesView{
			Workers: []query.AgentWorkerStatusView{{
				WorkerID:    "knowledge-worker-1",
				WorkerType:  "knowledge",
				Status:      "idle",
				LeaseActive: true,
			}},
			SideEffect: "none",
		}},
	})

	view, err := service.CheckAgentJobExternalLeaseReadiness(context.Background(), command.CheckAgentJobExternalLeaseReadinessCommand{})
	if err != nil {
		t.Fatalf("check readiness: %v", err)
	}
	if !view.Ready || view.Reason != "agent_job_external_lease_ready" {
		t.Fatalf("expected ready result-ack readiness: %#v", view)
	}
	if !view.ExternalLeaseReady || !view.AgentJobResultAckReady || !view.StrictLeaseTokenEnabled || !view.AgentJobWorkerReady {
		t.Fatalf("unexpected ready flags: %#v", view)
	}
	if view.ExecutionScope != "outbox_delivery_and_agent_job_result_ack" || len(view.Blockers) != 0 {
		t.Fatalf("unexpected ready detail: %#v", view)
	}
}

func TestAgentJobExternalLeaseReadinessBlocksDangerWorkerCoverage(t *testing.T) {
	service := NewAgentJobExternalLeaseReadinessService(AgentJobExternalLeaseReadinessDeps{
		QueueBackend: staticRuntimeQueueBackend{view: query.QueueBackendView{
			Provider:               "nats_jetstream",
			Mode:                   "external_lease",
			AgentJobExecutionOwner: "python_ai_worker_with_nats_result_ack",
			ExternalLease: &query.QueueExternalLeaseGate{
				AllowExecution:   true,
				ExecutionScope:   "outbox_delivery_and_agent_job_result_ack",
				AllowedWorkKinds: []string{"outbox_delivery", "agent_job"},
			},
		}},
		RuntimeConfig: staticRuntimeConfig{view: query.RuntimeConfigView{
			Workers: query.RuntimeWorkerConfigView{
				AgentJobStrictLeaseToken:         true,
				QueueExternalLeaseAgentJobEnable: true,
			},
			SideEffect: "none",
		}},
		AgentJobs: staticRuntimeAgentJobMetrics{view: query.AgentJobMetricsView{
			Pressure: query.AgentJobPressureMetricsView{
				ByType: []query.AgentJobTypePressureView{{
					JobType:      "rag_ingest",
					Pending:      15,
					HighPressure: true,
				}},
			},
		}},
		AgentWorkers: staticRuntimeAgentWorkers{view: query.AgentWorkerStatusesView{SideEffect: "none"}},
	})

	view, err := service.CheckAgentJobExternalLeaseReadiness(context.Background(), command.CheckAgentJobExternalLeaseReadinessCommand{})
	if err != nil {
		t.Fatalf("check readiness: %v", err)
	}
	if view.Ready || view.AgentJobWorkerReady {
		t.Fatalf("expected blocked worker coverage readiness: %#v", view)
	}
	if !containsString(view.Blockers, "agent_job_worker_coverage_blocked") {
		t.Fatalf("expected worker coverage blocker: %#v", view.Blockers)
	}
	if len(view.WorkerCoverage) != 1 || view.WorkerCoverage[0].CoverageStatus != "danger" {
		t.Fatalf("expected danger coverage detail: %#v", view.WorkerCoverage)
	}
}

func TestAgentJobExternalLeaseReadinessBlocksMissingResultAckGate(t *testing.T) {
	service := NewAgentJobExternalLeaseReadinessService(AgentJobExternalLeaseReadinessDeps{
		QueueBackend: staticRuntimeQueueBackend{view: query.QueueBackendView{
			Provider:               "nats_jetstream",
			Mode:                   "external_lease",
			AgentJobExecutionOwner: "python_ai_worker_state_store_lease",
			ExternalLease: &query.QueueExternalLeaseGate{
				AllowExecution:   true,
				ExecutionScope:   "outbox_delivery_only",
				AllowedWorkKinds: []string{"outbox_delivery"},
				BlockedWorkKinds: []query.QueueExternalLeaseBlock{{
					WorkKind: "agent_job",
					Reason:   "agent_job result-ack smoke missing",
				}},
			},
		}},
		RuntimeConfig: staticRuntimeConfig{view: query.RuntimeConfigView{
			Workers: query.RuntimeWorkerConfigView{
				AgentJobStrictLeaseToken: true,
			},
			SideEffect: "none",
		}},
		AgentJobs:    staticRuntimeAgentJobMetrics{view: query.AgentJobMetricsView{}},
		AgentWorkers: staticRuntimeAgentWorkers{view: query.AgentWorkerStatusesView{SideEffect: "none"}},
	})

	view, err := service.CheckAgentJobExternalLeaseReadiness(context.Background(), command.CheckAgentJobExternalLeaseReadinessCommand{})
	if err != nil {
		t.Fatalf("check readiness: %v", err)
	}
	if view.Ready || view.AgentJobResultAckReady {
		t.Fatalf("expected blocked result-ack readiness: %#v", view)
	}
	for _, blocker := range []string{
		"agent_job_not_allowed_in_external_lease",
		"agent_job_result_ack_owner_not_enabled",
		"queue_external_lease_agent_job_flag_disabled",
	} {
		if !containsString(view.Blockers, blocker) {
			t.Fatalf("expected blocker %s in %#v", blocker, view.Blockers)
		}
	}
	if len(view.BlockedWorkKinds) != 1 {
		t.Fatalf("expected blocked work kind detail: %#v", view.BlockedWorkKinds)
	}
}
