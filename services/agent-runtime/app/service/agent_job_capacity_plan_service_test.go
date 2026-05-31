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

func TestAgentJobCapacityPlanRecommendsRecoveryAndConcurrencyTuning(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	jobs := appservice.NewAgentJobService(store)
	metrics := appservice.NewAgentJobMetricsService(store, nil)
	workers := appservice.NewAgentWorkerStatusService()
	service := appservice.NewAgentJobCapacityPlanService(appservice.AgentJobCapacityPlanDeps{
		AgentJobs:    metrics,
		AgentWorkers: workers,
	})
	now := time.Date(2026, 5, 31, 15, 0, 0, 0, time.UTC)

	for index := 0; index < 10; index++ {
		cmd := sampleKnowledgeJobCommand(
			"group_memory_extract:qq:27234224:"+string(rune('a'+index)),
			"group_memory_extract",
			"27234224",
			map[string]string{"group_id": "27234224"},
			now.Add(time.Duration(index)*time.Second),
		)
		if _, err := jobs.Create(ctx, cmd); err != nil {
			t.Fatalf("create group memory pending job %d: %v", index, err)
		}
	}

	for index := 0; index < 5; index++ {
		cmd := sampleKnowledgeJobCommand(
			"image_generation:qq:1049511700:"+string(rune('a'+index)),
			"image_generation",
			"1049511700",
			map[string]string{"prompt": "fixture"},
			now.Add(time.Duration(index)*time.Second),
		)
		created, err := jobs.Create(ctx, cmd)
		if err != nil {
			t.Fatalf("create image job %d: %v", index, err)
		}
		leased, err := jobs.Lease(ctx, command.AgentJobLeaseCommand{
			JobID:      created.JobID,
			WorkerID:   "image-worker",
			LeaseToken: created.JobID + "-lease",
			TTLSeconds: 300,
			Timestamp:  now.Add(time.Duration(index) * time.Second),
		})
		if err != nil {
			t.Fatalf("lease image job %d: %v", index, err)
		}
		if _, err := jobs.MarkRunning(ctx, command.MarkAgentJobRunningCommand{
			JobID:      leased.JobID,
			LeaseToken: leased.LeaseToken,
			Timestamp:  now.Add(time.Duration(index)*time.Second + time.Second),
		}); err != nil {
			t.Fatalf("run image job %d: %v", index, err)
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
		Status:     "running",
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

	view, err := service.PlanAgentJobCapacity(ctx, command.PlanAgentJobCapacityCommand{
		JobLimit:          50,
		EventLimit:        50,
		StaleAfterSeconds: 900,
	})
	if err != nil {
		t.Fatalf("plan capacity: %v", err)
	}
	if view.Ready || view.Reason != "agent_job_capacity_attention_required" {
		t.Fatalf("expected attention-required plan: %+v", view)
	}
	if view.Summary.JobTypes != 3 ||
		view.Summary.HighPressureJobTypes != 2 ||
		view.Summary.CapacityBlockedJobTypes != 1 ||
		view.Summary.WorkerWarningJobTypes != 2 ||
		view.Summary.ActiveWorkerJobTypes != 1 ||
		view.Summary.FailedWorkerJobTypes != 1 {
		t.Fatalf("unexpected summary: %+v", view.Summary)
	}
	if !capacityPlanContains(view.Blockers, "agent_job_capacity_blocked") ||
		!capacityPlanContains(view.Blockers, "agent_job_high_pressure") ||
		!capacityPlanContains(view.Blockers, "agent_job_worker_warning") {
		t.Fatalf("missing blockers: %+v", view.Blockers)
	}

	group := findCapacityPlanItem(t, view, "group_memory_extract")
	if group.Severity != "danger" ||
		group.Action != "start_or_recover_python_worker" ||
		group.Coverage.CoverageReason != "high_pressure_no_registered_worker" {
		t.Fatalf("unexpected group memory item: %+v", group)
	}
	image := findCapacityPlanItem(t, view, "image_generation")
	if image.Severity != "warn" ||
		image.Action != "increase_worker_concurrency_or_prioritize_queue" ||
		image.Coverage.ActiveWorkers != 1 {
		t.Fatalf("unexpected image item: %+v", image)
	}
	ragEval := findCapacityPlanItem(t, view, "rag_eval")
	if ragEval.Severity != "warn" ||
		ragEval.Action != "inspect_failed_worker" ||
		ragEval.Coverage.FailedWorkers != 1 {
		t.Fatalf("unexpected rag eval item: %+v", ragEval)
	}
	if view.SideEffect != "none" || view.Attributes["side_effect"] != "none" {
		t.Fatalf("capacity plan must be read-only: %+v", view)
	}
}

func TestAgentJobCapacityPlanReportsReadyWhenCoverageIsHealthy(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	jobs := appservice.NewAgentJobService(store)
	metrics := appservice.NewAgentJobMetricsService(store, nil)
	workers := appservice.NewAgentWorkerStatusService()
	service := appservice.NewAgentJobCapacityPlanService(appservice.AgentJobCapacityPlanDeps{
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

	view, err := service.PlanAgentJobCapacity(ctx, command.PlanAgentJobCapacityCommand{})
	if err != nil {
		t.Fatalf("plan capacity: %v", err)
	}
	if !view.Ready || view.Reason != "agent_job_capacity_ready" {
		t.Fatalf("expected ready plan: %+v", view)
	}
	item := findCapacityPlanItem(t, view, "image_generation")
	if item.Severity != "ok" || item.Action != "monitor" {
		t.Fatalf("unexpected ready item: %+v", item)
	}
	if len(view.Blockers) != 0 {
		t.Fatalf("unexpected blockers: %+v", view.Blockers)
	}
}

func findCapacityPlanItem(
	t *testing.T,
	view query.AgentJobCapacityPlanView,
	jobType string,
) query.AgentJobCapacityPlanItemView {
	t.Helper()
	for _, item := range view.Items {
		if item.JobType == jobType {
			return item
		}
	}
	t.Fatalf("capacity plan item not found for %s: %+v", jobType, view.Items)
	return query.AgentJobCapacityPlanItemView{}
}

func capacityPlanContains(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
