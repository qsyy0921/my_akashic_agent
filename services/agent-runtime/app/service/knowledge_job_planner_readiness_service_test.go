package service

import (
	"context"
	"testing"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

func TestKnowledgeJobPlannerReadinessAllowsDisabledPlannerPreflight(t *testing.T) {
	service := NewKnowledgeJobPlannerReadinessService(KnowledgeJobPlannerReadinessDeps{
		Previewer: staticKnowledgeJobPlannerPreview{view: query.KnowledgeJobPlannerPreviewView{
			Targets:         1,
			TotalJobs:       2,
			GroupMemoryJobs: 1,
			RagIngestJobs:   1,
			Groups:          []string{"27234224"},
			SideEffect:      "none",
		}},
		RuntimeConfig: staticRuntimeConfig{view: query.RuntimeConfigView{
			Workers: query.RuntimeWorkerConfigView{KnowledgeJobPlannerEnabled: false},
		}},
		RuntimeWorkers: staticRuntimeWorkerDiagnostics{view: query.RuntimeWorkerDiagnosticsView{
			Workers: []query.RuntimeWorkerView{{Name: "knowledge_job_planner", Enabled: false, Running: false}},
		}},
		AgentWorkers: staticRuntimeAgentWorkers{view: query.AgentWorkerStatusesView{
			Workers: []query.AgentWorkerStatusView{{
				WorkerID:    "knowledge-worker-1",
				WorkerType:  "knowledge",
				Status:      "idle",
				LeaseActive: true,
			}},
		}},
	})

	view, err := service.CheckKnowledgeJobPlannerReadiness(context.Background(), command.CheckKnowledgeJobPlannerReadinessCommand{})
	if err != nil {
		t.Fatalf("readiness: %v", err)
	}
	if !view.Ready || view.Reason != "knowledge_job_planner_ready" || len(view.Blockers) != 0 {
		t.Fatalf("expected ready preflight, got %#v", view)
	}
	if view.PlannerEnabled || view.PlannerRunning || !view.KnowledgeWorkerReady || view.KnowledgeWorkerActive != 1 {
		t.Fatalf("unexpected readiness flags: %#v", view)
	}
}

func TestKnowledgeJobPlannerReadinessReportsAdmissionBlockers(t *testing.T) {
	service := NewKnowledgeJobPlannerReadinessService(KnowledgeJobPlannerReadinessDeps{
		Previewer: staticKnowledgeJobPlannerPreview{view: query.KnowledgeJobPlannerPreviewView{
			Targets:    0,
			TotalJobs:  0,
			SideEffect: "none",
		}},
		RuntimeConfig: staticRuntimeConfig{view: query.RuntimeConfigView{
			Workers: query.RuntimeWorkerConfigView{KnowledgeJobPlannerEnabled: true},
		}},
		RuntimeWorkers: staticRuntimeWorkerDiagnostics{view: query.RuntimeWorkerDiagnosticsView{
			Workers: []query.RuntimeWorkerView{{Name: "knowledge_job_planner", Enabled: true, Running: false}},
		}},
		AgentWorkers: staticRuntimeAgentWorkers{view: query.AgentWorkerStatusesView{
			Workers: []query.AgentWorkerStatusView{
				{WorkerID: "knowledge-worker-stale", WorkerType: "knowledge", Status: "running", Stale: true},
				{WorkerID: "knowledge-worker-failed", WorkerType: "knowledge", Status: "failed"},
				{WorkerID: "knowledge-worker-stopped", WorkerType: "knowledge", Status: "stopped"},
			},
		}},
	})

	view, err := service.CheckKnowledgeJobPlannerReadiness(context.Background(), command.CheckKnowledgeJobPlannerReadinessCommand{})
	if err != nil {
		t.Fatalf("readiness: %v", err)
	}
	if view.Ready || view.Reason != "knowledge_job_planner_not_ready" {
		t.Fatalf("expected blocked readiness, got %#v", view)
	}
	expected := []string{
		"knowledge_job_planner_no_planned_jobs",
		"knowledge_job_planner_worker_not_running",
		"knowledge_worker_unavailable",
	}
	for index, item := range expected {
		if len(view.Blockers) <= index || view.Blockers[index] != item {
			t.Fatalf("unexpected blockers: %#v", view.Blockers)
		}
	}
	if view.KnowledgeWorkerActive != 0 || view.KnowledgeWorkerStale != 1 || view.KnowledgeWorkerFailed != 1 || view.KnowledgeWorkerStopped != 1 {
		t.Fatalf("unexpected worker counts: %#v", view)
	}
}
