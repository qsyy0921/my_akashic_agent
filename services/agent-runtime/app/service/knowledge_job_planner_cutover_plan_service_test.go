package service

import (
	"context"
	"testing"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

func TestKnowledgeJobPlannerCutoverPlanRecommendsGoPlanner(t *testing.T) {
	service := NewKnowledgeJobPlannerCutoverPlanService(KnowledgeJobPlannerCutoverPlanDeps{
		Readiness: staticKnowledgeJobPlannerReadiness{view: query.KnowledgeJobPlannerReadinessView{
			Ready:                 true,
			Reason:                "knowledge_job_planner_ready",
			PlannerEnabled:        false,
			PlannerRunning:        false,
			KnowledgeWorkerReady:  true,
			KnowledgeWorkerActive: 1,
			Preview:               query.KnowledgeJobPlannerPreviewView{TotalJobs: 2, Targets: 1, SideEffect: "none"},
			SideEffect:            "none",
		}},
	})

	view, err := service.PlanKnowledgeJobPlannerCutover(context.Background(), command.PlanKnowledgeJobPlannerCutoverCommand{})
	if err != nil {
		t.Fatalf("plan cutover: %v", err)
	}
	if view.Ready || view.Decision != "ready_to_enable_go_planner" {
		t.Fatalf("expected ready-to-enable plan, got %#v", view)
	}
	if view.CurrentAdmissionOwner != "python_legacy_knowledge_enqueue" ||
		view.DesiredAdmissionOwner != "go_runtime_knowledge_job_planner" ||
		view.RecommendedAdmissionOwner != "go_runtime_knowledge_job_planner" {
		t.Fatalf("unexpected owners: %#v", view)
	}
	if len(view.Blockers) != 0 || view.SideEffect != "none" || len(view.EnableSteps) == 0 || len(view.RollbackSteps) == 0 {
		t.Fatalf("unexpected cutover metadata: %#v", view)
	}
}

func TestKnowledgeJobPlannerCutoverPlanBlocksWhenReadinessFails(t *testing.T) {
	service := NewKnowledgeJobPlannerCutoverPlanService(KnowledgeJobPlannerCutoverPlanDeps{
		Readiness: staticKnowledgeJobPlannerReadiness{view: query.KnowledgeJobPlannerReadinessView{
			Ready:          false,
			Reason:         "knowledge_job_planner_not_ready",
			PlannerEnabled: true,
			PlannerRunning: false,
			Blockers:       []string{"knowledge_worker_unavailable"},
			SideEffect:     "none",
		}},
	})

	view, err := service.PlanKnowledgeJobPlannerCutover(context.Background(), command.PlanKnowledgeJobPlannerCutoverCommand{})
	if err != nil {
		t.Fatalf("plan cutover: %v", err)
	}
	if view.Ready || view.Decision != "blocked" {
		t.Fatalf("expected blocked plan, got %#v", view)
	}
	expected := []string{"knowledge_job_planner_readiness_not_ready", "knowledge_worker_unavailable"}
	for index, item := range expected {
		if len(view.Blockers) <= index || view.Blockers[index] != item {
			t.Fatalf("unexpected blockers: %#v", view.Blockers)
		}
	}
}

func TestKnowledgeJobPlannerCutoverPlanSupportsPythonRollback(t *testing.T) {
	service := NewKnowledgeJobPlannerCutoverPlanService(KnowledgeJobPlannerCutoverPlanDeps{
		Readiness: staticKnowledgeJobPlannerReadiness{view: query.KnowledgeJobPlannerReadinessView{
			Ready:          true,
			PlannerEnabled: true,
			PlannerRunning: true,
			SideEffect:     "none",
		}},
	})

	view, err := service.PlanKnowledgeJobPlannerCutover(context.Background(), command.PlanKnowledgeJobPlannerCutoverCommand{
		DesiredAdmissionOwner: "python_legacy",
	})
	if err != nil {
		t.Fatalf("plan rollback: %v", err)
	}
	if view.Ready || view.Decision != "ready_to_rollback_to_python_legacy" {
		t.Fatalf("expected rollback plan, got %#v", view)
	}
	if view.DesiredAdmissionOwner != "python_legacy_knowledge_enqueue" || len(view.Blockers) != 0 {
		t.Fatalf("unexpected rollback owners/blockers: %#v", view)
	}
}
