package service

import (
	"context"
	"testing"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type staticAgentJobExternalLeaseCutoverDiffBundleViewer struct {
	view query.AgentJobExternalLeaseLauncherBundleView
}

func (s staticAgentJobExternalLeaseCutoverDiffBundleViewer) GetAgentJobExternalLeaseLauncherBundle(context.Context, query.AgentJobExternalLeaseLauncherBundleFilter) (query.AgentJobExternalLeaseLauncherBundleView, error) {
	return s.view, nil
}

type staticAgentJobExternalLeaseCutoverDiffRuntimeConfigViewer struct {
	view query.RuntimeConfigView
}

func (s staticAgentJobExternalLeaseCutoverDiffRuntimeConfigViewer) GetRuntimeConfig(context.Context) (query.RuntimeConfigView, error) {
	return s.view, nil
}

type staticAgentJobExternalLeaseCutoverDiffQueueBackendViewer struct {
	view query.QueueBackendView
}

func (s staticAgentJobExternalLeaseCutoverDiffQueueBackendViewer) Get(context.Context) (query.QueueBackendView, error) {
	return s.view, nil
}

type staticAgentJobExternalLeaseCutoverDiffQueueTopologyViewer struct {
	view query.QueueTopologyView
}

func (s staticAgentJobExternalLeaseCutoverDiffQueueTopologyViewer) GetQueueTopology(context.Context) (query.QueueTopologyView, error) {
	return s.view, nil
}

func TestAgentJobExternalLeaseCutoverDiffReportsRuntimeDrift(t *testing.T) {
	service := NewAgentJobExternalLeaseCutoverDiffService(
		staticAgentJobExternalLeaseCutoverDiffBundleViewer{
			view: query.AgentJobExternalLeaseLauncherBundleView{
				Ready:                     false,
				Reason:                    "agent_job_external_lease_launcher_bundle_blocked",
				Blockers:                  []string{"agent_job_external_lease_readiness_not_ready"},
				DesiredExecutionOwner:     "python_ai_worker_with_nats_result_ack",
				RecommendedExecutionOwner: "python_ai_worker_with_nats_result_ack",
				CurrentExecutionOwner:     "python_ai_worker_state_store_lease",
				LauncherParameters: map[string]string{
					"QueueBackend":                      "nats_jetstream",
					"QueueMode":                         "external_lease",
					"AgentJobStrictLeaseToken":          "true",
					"QueueExternalLeaseCutover":         "true",
					"QueueExternalLeaseAgentJobEnabled": "true",
				},
				EnvironmentOverrides: map[string]string{
					"AKASHIC_QUEUE_BACKEND":                          "nats_jetstream",
					"AKASHIC_QUEUE_MODE":                             "external_lease",
					"AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN":           "true",
					"AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER":           "true",
					"AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED": "true",
				},
				RequiredExternalInputs: []query.AgentJobExternalLeaseLauncherBundleInputView{{
					Parameter:      "QueueDSN",
					EnvironmentKey: "AKASHIC_QUEUE_DSN",
					Required:       true,
					Secret:         true,
				}},
				SideEffect: "none",
			},
		},
		staticAgentJobExternalLeaseCutoverDiffRuntimeConfigViewer{
			view: query.RuntimeConfigView{
				Environment: []query.RuntimeEnvVarView{
					{Key: "AKASHIC_QUEUE_MODE", Present: true, ValueRedacted: "local_state_store"},
					{Key: "AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN", Present: true, ValueRedacted: "false"},
				},
			},
		},
		staticAgentJobExternalLeaseCutoverDiffQueueBackendViewer{
			view: query.QueueBackendView{
				Provider:               "local",
				Mode:                   "local_state_store",
				DSNConfigured:          false,
				AgentJobExecutionOwner: "python_ai_worker_state_store_lease",
			},
		},
		staticAgentJobExternalLeaseCutoverDiffQueueTopologyViewer{
			view: query.QueueTopologyView{
				ExternalLeaseReady: false,
				WorkKinds: []query.QueueTopologyWorkKind{{
					WorkKind:       "agent_job",
					ExecutionOwner: "python_ai_worker_state_store_lease",
					AckOwner:       "go_state_store_api",
				}},
			},
		},
	)

	view, err := service.GetAgentJobExternalLeaseCutoverDiff(context.Background(), query.AgentJobExternalLeaseCutoverDiffFilter{
		DesiredExecutionOwner: "python_ai_worker_with_nats_result_ack",
	})
	if err != nil {
		t.Fatalf("cutover diff returned error: %v", err)
	}
	if view.Ready || view.Reason != "agent_job_external_lease_cutover_diff_blocked" {
		t.Fatalf("expected blocked cutover diff, got %+v", view)
	}
	if len(view.Drift) == 0 {
		t.Fatalf("expected drift items, got %+v", view)
	}
	if view.CurrentExecutionOwner != "python_ai_worker_state_store_lease" || view.ExpectedAckOwner != "nats_external_lease_result_ack" {
		t.Fatalf("unexpected owner fields: %+v", view)
	}
}

func TestAgentJobExternalLeaseCutoverDiffBecomesReadyWhenRuntimeMatchesBundle(t *testing.T) {
	service := NewAgentJobExternalLeaseCutoverDiffService(
		staticAgentJobExternalLeaseCutoverDiffBundleViewer{
			view: query.AgentJobExternalLeaseLauncherBundleView{
				Ready:                     true,
				Reason:                    "agent_job_external_lease_launcher_bundle_ready",
				DesiredExecutionOwner:     "python_ai_worker_with_nats_result_ack",
				RecommendedExecutionOwner: "python_ai_worker_with_nats_result_ack",
				CurrentExecutionOwner:     "python_ai_worker_with_nats_result_ack",
				LauncherParameters: map[string]string{
					"QueueBackend": "nats_jetstream",
					"QueueMode":    "external_lease",
				},
				EnvironmentOverrides: map[string]string{
					"AKASHIC_QUEUE_BACKEND":                "nats_jetstream",
					"AKASHIC_QUEUE_MODE":                   "external_lease",
					"AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN": "true",
				},
				RequiredExternalInputs: []query.AgentJobExternalLeaseLauncherBundleInputView{{
					Parameter:      "QueueDSN",
					EnvironmentKey: "AKASHIC_QUEUE_DSN",
					Required:       true,
					Secret:         true,
				}},
				SideEffect: "none",
			},
		},
		staticAgentJobExternalLeaseCutoverDiffRuntimeConfigViewer{
			view: query.RuntimeConfigView{
				Environment: []query.RuntimeEnvVarView{
					{Key: "AKASHIC_QUEUE_BACKEND", Present: true, ValueRedacted: "nats_jetstream"},
					{Key: "AKASHIC_QUEUE_MODE", Present: true, ValueRedacted: "external_lease"},
					{Key: "AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN", Present: true, ValueRedacted: "true"},
				},
			},
		},
		staticAgentJobExternalLeaseCutoverDiffQueueBackendViewer{
			view: query.QueueBackendView{
				Provider:               "nats_jetstream",
				Mode:                   "external_lease",
				DSNConfigured:          true,
				AgentJobExecutionOwner: "python_ai_worker_with_nats_result_ack",
			},
		},
		staticAgentJobExternalLeaseCutoverDiffQueueTopologyViewer{
			view: query.QueueTopologyView{
				ExternalLeaseReady: true,
				WorkKinds: []query.QueueTopologyWorkKind{{
					WorkKind:       "agent_job",
					ExecutionOwner: "python_ai_worker_with_nats_result_ack",
					AckOwner:       "nats_external_lease_result_ack",
				}},
			},
		},
	)

	view, err := service.GetAgentJobExternalLeaseCutoverDiff(context.Background(), query.AgentJobExternalLeaseCutoverDiffFilter{
		DesiredExecutionOwner: "python_ai_worker_with_nats_result_ack",
	})
	if err != nil {
		t.Fatalf("cutover diff returned error: %v", err)
	}
	if !view.Ready || view.Reason != "agent_job_external_lease_cutover_diff_ready" {
		t.Fatalf("expected ready cutover diff, got %+v", view)
	}
	if len(view.Drift) != 0 {
		t.Fatalf("expected no drift, got %+v", view.Drift)
	}
	if len(view.Matching) == 0 {
		t.Fatalf("expected matching items, got %+v", view)
	}
}
