package service

import (
	"context"
	"testing"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

func TestAgentJobExternalLeaseLauncherBundleBuildsNATSEnableBundle(t *testing.T) {
	service := NewAgentJobExternalLeaseLauncherBundleService(staticAgentJobExternalLeaseLauncherBundlePlanner{
		view: query.AgentJobExternalLeasePlanView{
			Ready:                     false,
			Decision:                  "ready_to_enable_result_ack",
			DesiredExecutionOwner:     "python_ai_worker_with_nats_result_ack",
			RecommendedExecutionOwner: "python_ai_worker_with_nats_result_ack",
			CurrentExecutionOwner:     "python_ai_worker_state_store_lease",
			SideEffect:                "none",
		},
	})

	view, err := service.GetAgentJobExternalLeaseLauncherBundle(context.Background(), query.AgentJobExternalLeaseLauncherBundleFilter{})
	if err != nil {
		t.Fatalf("get launcher bundle: %v", err)
	}
	if view.Reason != "agent_job_external_lease_launcher_bundle_ready" || !view.Ready {
		t.Fatalf("unexpected launcher bundle readiness: %+v", view)
	}
	if view.ScriptPath != ".\\scripts\\start-agent-runtime.ps1" {
		t.Fatalf("unexpected script path: %+v", view)
	}
	for key, expected := range map[string]string{
		"QueueBackend":                      "nats_jetstream",
		"QueueMode":                         "external_lease",
		"QueueExternalLeaseCutover":         "true",
		"QueueDualReadSmokePassed":          "true",
		"QueueStateLeaseWorkersDisabled":    "true",
		"QueueExternalLeaseAgentJobEnabled": "true",
		"QueueAgentJobDuplicateSmokePassed": "true",
		"QueueAgentJobFlowSmokePassed":      "true",
		"AgentJobStrictLeaseToken":          "true",
	} {
		if view.LauncherParameters[key] != expected {
			t.Fatalf("missing launcher parameter %s=%s: %+v", key, expected, view.LauncherParameters)
		}
	}
	if view.EnvironmentOverrides["AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED"] != "true" ||
		view.EnvironmentOverrides["AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN"] != "true" {
		t.Fatalf("unexpected env overrides: %+v", view.EnvironmentOverrides)
	}
	if len(view.RequiredExternalInputs) != 1 ||
		view.RequiredExternalInputs[0].Parameter != "QueueDSN" ||
		!view.RequiredExternalInputs[0].Required ||
		!view.RequiredExternalInputs[0].Secret {
		t.Fatalf("unexpected required external inputs: %+v", view.RequiredExternalInputs)
	}
}

func TestAgentJobExternalLeaseLauncherBundleReportsBlockedPlan(t *testing.T) {
	service := NewAgentJobExternalLeaseLauncherBundleService(staticAgentJobExternalLeaseLauncherBundlePlanner{
		view: query.AgentJobExternalLeasePlanView{
			Ready:                     false,
			Decision:                  "blocked",
			DesiredExecutionOwner:     "python_ai_worker_with_nats_result_ack",
			RecommendedExecutionOwner: "python_ai_worker_with_nats_result_ack",
			CurrentExecutionOwner:     "python_ai_worker_state_store_lease",
			Blockers:                  []string{"agent_job_external_lease_readiness_not_ready"},
			SideEffect:                "none",
		},
	})

	view, err := service.GetAgentJobExternalLeaseLauncherBundle(context.Background(), query.AgentJobExternalLeaseLauncherBundleFilter{})
	if err != nil {
		t.Fatalf("get blocked launcher bundle: %v", err)
	}
	if view.Ready {
		t.Fatalf("blocked plan must not report ready launcher bundle: %+v", view)
	}
	if view.Reason != "agent_job_external_lease_launcher_bundle_blocked" {
		t.Fatalf("unexpected blocked reason: %+v", view)
	}
	if !containsString(view.Blockers, "agent_job_external_lease_readiness_not_ready") {
		t.Fatalf("missing plan blocker: %+v", view.Blockers)
	}
	if view.LauncherParameters["QueueBackend"] != "nats_jetstream" {
		t.Fatalf("blocked bundle should still emit canonical launcher parameters: %+v", view.LauncherParameters)
	}
}

type staticAgentJobExternalLeaseLauncherBundlePlanner struct {
	view query.AgentJobExternalLeasePlanView
}

func (s staticAgentJobExternalLeaseLauncherBundlePlanner) PlanAgentJobExternalLease(context.Context, command.PlanAgentJobExternalLeaseCommand) (query.AgentJobExternalLeasePlanView, error) {
	return s.view, nil
}
