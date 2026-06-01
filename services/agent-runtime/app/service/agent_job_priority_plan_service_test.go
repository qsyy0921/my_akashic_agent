package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/memory"
)

func TestAgentJobPriorityPlanOrdersBackpressureBeforeHealthyWork(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	jobs := appservice.NewAgentJobService(store)
	metrics := appservice.NewAgentJobMetricsService(store, nil)
	workers := appservice.NewAgentWorkerStatusService()
	service := appservice.NewAgentJobPriorityPlanService(appservice.AgentJobPriorityPlanDeps{
		AgentJobs:    metrics,
		AgentWorkers: workers,
	})
	now := time.Now().UTC().Add(-2 * time.Minute)

	for index := 0; index < 10; index++ {
		if _, err := jobs.Create(ctx, sampleKnowledgeJobCommand(
			"group_memory_extract:qq:27234224:"+string(rune('a'+index)),
			"group_memory_extract",
			"27234224",
			map[string]string{"group_id": "27234224"},
			now.Add(time.Duration(index)*time.Second),
		)); err != nil {
			t.Fatalf("create group memory job %d: %v", index, err)
		}
	}
	for index := 0; index < 3; index++ {
		if _, err := jobs.Create(ctx, sampleKnowledgeJobCommand(
			"image_generation:qq:1049511700:"+string(rune('a'+index)),
			"image_generation",
			"1049511700",
			map[string]string{"prompt": "fixture"},
			now.Add(time.Duration(index)*time.Second),
		)); err != nil {
			t.Fatalf("create image job %d: %v", index, err)
		}
	}
	if _, err := jobs.Create(ctx, sampleKnowledgeJobCommand(
		"rag_eval:qq:27234224:fixture",
		"rag_eval",
		"27234224",
		map[string]string{"dataset_id": "fixture"},
		now,
	)); err != nil {
		t.Fatalf("create rag eval job: %v", err)
	}
	if _, err := workers.ReportAgentWorkerStatus(ctx, command.ReportAgentWorkerStatusCommand{
		WorkerID:   "image-worker",
		WorkerType: "image_generation",
		Status:     "idle",
		Source:     "python",
		Timestamp:  time.Now().UTC(),
	}); err != nil {
		t.Fatalf("report image worker: %v", err)
	}
	if _, err := workers.ReportAgentWorkerStatus(ctx, command.ReportAgentWorkerStatusCommand{
		WorkerID:   "rag-eval-worker",
		WorkerType: "rag_eval",
		Status:     "failed",
		LastError:  "fixture failure",
		Source:     "python",
		Timestamp:  time.Now().UTC(),
	}); err != nil {
		t.Fatalf("report rag eval worker: %v", err)
	}

	view, err := service.PlanAgentJobPriority(ctx, command.PlanAgentJobPriorityCommand{
		JobLimit:          50,
		EventLimit:        50,
		StaleAfterSeconds: 900,
	})
	if err != nil {
		t.Fatalf("plan priority: %v", err)
	}
	if view.Ready || view.Reason != "agent_job_priority_attention_required" {
		t.Fatalf("expected attention-required plan: %+v", view)
	}
	if view.Summary.JobTypes != 3 ||
		view.Summary.HighPriorityJobTypes != 2 ||
		view.Summary.BlockedJobTypes != 1 ||
		view.Summary.WarningJobTypes != 1 ||
		view.Summary.MaxPriorityScore != 100 {
		t.Fatalf("unexpected summary: %+v", view.Summary)
	}
	if len(view.Items) != 3 {
		t.Fatalf("expected 3 priority items, got %+v", view.Items)
	}
	if view.Items[0].JobType != "group_memory_extract" ||
		view.Items[0].Rank != 1 ||
		view.Items[0].PriorityClass != "critical" ||
		view.Items[0].Action != "recover_worker_before_priority_tuning" {
		t.Fatalf("unexpected first priority item: %+v", view.Items[0])
	}
	ragEval := findPriorityPlanItem(t, view, "rag_eval")
	if ragEval.PriorityClass != "high" || ragEval.PriorityScore != 90 {
		t.Fatalf("unexpected rag eval priority: %+v", ragEval)
	}
	image := findPriorityPlanItem(t, view, "image_generation")
	if image.PriorityClass != "low" || image.Action != "monitor" {
		t.Fatalf("unexpected image priority: %+v", image)
	}
	if view.SideEffect != "none" || view.Attributes["side_effect"] != "none" ||
		view.Attributes["worker_control_owner"] != "python" {
		t.Fatalf("priority plan must be read-only and Python-owned for execution: %+v", view)
	}
}

func TestAgentJobPriorityPlanReportsReadyWhenCoverageIsHealthy(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	jobs := appservice.NewAgentJobService(store)
	metrics := appservice.NewAgentJobMetricsService(store, nil)
	workers := appservice.NewAgentWorkerStatusService()
	service := appservice.NewAgentJobPriorityPlanService(appservice.AgentJobPriorityPlanDeps{
		AgentJobs:    metrics,
		AgentWorkers: workers,
	})

	if _, err := jobs.Create(ctx, sampleKnowledgeJobCommand(
		"image_generation:qq:1049511700:single",
		"image_generation",
		"1049511700",
		map[string]string{"prompt": "fixture"},
		time.Now().UTC(),
	)); err != nil {
		t.Fatalf("create image job: %v", err)
	}
	if _, err := workers.ReportAgentWorkerStatus(ctx, command.ReportAgentWorkerStatusCommand{
		WorkerID:   "image-worker",
		WorkerType: "image_generation",
		Status:     "idle",
		Source:     "python",
		Timestamp:  time.Now().UTC(),
	}); err != nil {
		t.Fatalf("report image worker: %v", err)
	}

	view, err := service.PlanAgentJobPriority(ctx, command.PlanAgentJobPriorityCommand{})
	if err != nil {
		t.Fatalf("plan priority: %v", err)
	}
	if !view.Ready || view.Reason != "agent_job_priority_ready" {
		t.Fatalf("expected ready plan: %+v", view)
	}
	item := findPriorityPlanItem(t, view, "image_generation")
	if item.PriorityClass != "low" || item.PriorityScore != 20 {
		t.Fatalf("unexpected ready priority item: %+v", item)
	}
	if len(view.Blockers) != 0 {
		t.Fatalf("unexpected blockers: %+v", view.Blockers)
	}
}

func findPriorityPlanItem(
	t *testing.T,
	view query.AgentJobPriorityPlanView,
	jobType string,
) query.AgentJobPriorityPlanItemView {
	t.Helper()
	for _, item := range view.Items {
		if item.JobType == jobType {
			return item
		}
	}
	t.Fatalf("priority plan item not found for %s: %+v", jobType, view.Items)
	return query.AgentJobPriorityPlanItemView{}
}
