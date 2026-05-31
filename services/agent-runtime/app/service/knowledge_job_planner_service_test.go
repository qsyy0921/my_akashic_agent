package service

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/memory"
)

func TestKnowledgeJobPlannerCreatesObserveOnlyGroupJobs(t *testing.T) {
	ctx := context.Background()
	observeTargets := NewObserveTargetService()
	syncObserveTargetsForPlanner(t, observeTargets)
	agentJobs := NewAgentJobService(memory.NewStore())
	planner := NewKnowledgeJobPlannerService(observeTargets, agentJobs)
	now := time.Date(2026, 5, 31, 8, 2, 0, 0, time.UTC)

	view, err := planner.PlanKnowledgeJobs(ctx, command.PlanKnowledgeJobsCommand{
		PlannerID:       "planner-a",
		AgentID:         "knowledge-worker",
		IntervalSeconds: 60,
		MaxAttempts:     2,
		RagMaxMessages:  500,
		RagParse:        true,
		Timestamp:       now,
	})
	if err != nil {
		t.Fatalf("plan knowledge jobs: %v", err)
	}

	if view.Targets != 1 || view.SkippedTargets != 3 || view.GroupMemoryJobs != 1 || view.RagIngestJobs != 2 {
		t.Fatalf("unexpected planner summary: %#v", view)
	}
	if view.CreatedOrExisting != 3 || view.SuppressedByDedupe != 0 {
		t.Fatalf("unexpected create/dedupe summary: %#v", view)
	}
	if len(view.Groups) != 1 || view.Groups[0] != "27234224" {
		t.Fatalf("unexpected groups: %#v", view.Groups)
	}

	jobs, err := agentJobs.List(ctx, query.AgentJobFilter{Limit: 10})
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(jobs) != 3 {
		t.Fatalf("expected three jobs, got %d: %#v", len(jobs), jobs)
	}
	groupMemory := findPlannedJob(t, jobs, "group_memory_extract")
	if groupMemory.AgentID != "knowledge-worker" ||
		groupMemory.Route.AccountID != "1049511700" ||
		groupMemory.Route.ConversationID != "27234224" ||
		groupMemory.Payload["session_key"] != "qq:gqq:27234224" ||
		groupMemory.Payload["observe_only"] != "true" ||
		groupMemory.Metadata["scheduler"] != "planner-a" {
		t.Fatalf("unexpected group memory job: %#v", groupMemory)
	}
	rag := findPlannedDatasetJob(t, jobs, "ds-guides")
	if rag.Payload["max_messages"] != "500" ||
		rag.Payload["parse"] != "true" ||
		rag.Metadata["dedupe_key"] != "knowledge:rag_ingest:qq:1049511700:27234224:ds-guides" {
		t.Fatalf("unexpected rag ingest job: %#v", rag)
	}
}

func TestKnowledgeJobPlannerPreviewIsReadOnly(t *testing.T) {
	ctx := context.Background()
	observeTargets := NewObserveTargetService()
	syncObserveTargetsForPlanner(t, observeTargets)
	agentJobs := NewAgentJobService(memory.NewStore())
	planner := NewKnowledgeJobPlannerService(observeTargets, agentJobs)
	now := time.Date(2026, 5, 31, 8, 2, 0, 0, time.UTC)

	view, err := planner.PreviewKnowledgeJobs(ctx, command.PlanKnowledgeJobsCommand{
		PlannerID:       "planner-a",
		AgentID:         "knowledge-worker",
		IntervalSeconds: 60,
		MaxAttempts:     2,
		RagMaxMessages:  500,
		RagParse:        true,
		Timestamp:       now,
	})
	if err != nil {
		t.Fatalf("preview knowledge jobs: %v", err)
	}

	if view.SideEffect != "none" || view.IntervalSeconds != 60 || view.TotalJobs != 3 {
		t.Fatalf("unexpected preview summary: %#v", view)
	}
	if view.Targets != 1 || view.SkippedTargets != 3 || view.GroupMemoryJobs != 1 || view.RagIngestJobs != 2 {
		t.Fatalf("unexpected target/job counts: %#v", view)
	}
	if len(view.Plans) != 1 || len(view.Plans[0].Jobs) != 3 {
		t.Fatalf("unexpected target plans: %#v", view.Plans)
	}
	firstJob := view.Plans[0].Jobs[0]
	if firstJob.JobID != "group_memory_extract:qq:27234224:29670242" ||
		firstJob.JobType != "group_memory_extract" ||
		firstJob.DedupeKey != "knowledge:group_memory_extract:qq:1049511700:27234224" ||
		firstJob.Payload["observe_only"] != "true" ||
		firstJob.Metadata["scheduler"] != "planner-a" {
		t.Fatalf("unexpected group memory job plan: %#v", firstJob)
	}
	if len(view.Skipped) != 3 || view.Skipped[0].Reason == "" {
		t.Fatalf("expected skipped target reasons, got %#v", view.Skipped)
	}

	jobs, err := agentJobs.List(ctx, query.AgentJobFilter{Limit: 10})
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("preview must not create jobs, got %#v", jobs)
	}
}

func TestKnowledgeJobPlannerReportsDedupeSuppressionAcrossBuckets(t *testing.T) {
	ctx := context.Background()
	observeTargets := NewObserveTargetService()
	syncObserveTargetsForPlanner(t, observeTargets)
	agentJobs := NewAgentJobService(memory.NewStore())
	planner := NewKnowledgeJobPlannerService(observeTargets, agentJobs)
	first := time.Date(2026, 5, 31, 8, 2, 0, 0, time.UTC)
	second := first.Add(time.Minute)

	if _, err := planner.PlanKnowledgeJobs(ctx, command.PlanKnowledgeJobsCommand{
		PlannerID:       "planner-a",
		AgentID:         "knowledge-worker",
		IntervalSeconds: 60,
		Timestamp:       first,
		RagParse:        true,
	}); err != nil {
		t.Fatalf("first plan: %v", err)
	}
	view, err := planner.PlanKnowledgeJobs(ctx, command.PlanKnowledgeJobsCommand{
		PlannerID:       "planner-a",
		AgentID:         "knowledge-worker",
		IntervalSeconds: 60,
		Timestamp:       second,
		RagParse:        true,
	})
	if err != nil {
		t.Fatalf("second plan: %v", err)
	}

	if view.CreatedOrExisting != 3 || view.SuppressedByDedupe != 3 {
		t.Fatalf("expected active dedupe suppression on next bucket, got %#v", view)
	}
	jobs, err := agentJobs.List(ctx, query.AgentJobFilter{Limit: 10})
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(jobs) != 3 {
		t.Fatalf("dedupe should keep active job count at three, got %d", len(jobs))
	}
}

func syncObserveTargetsForPlanner(t *testing.T, observeTargets *ObserveTargetService) {
	t.Helper()
	_, err := observeTargets.SyncObserveTargets(context.Background(), command.SyncObserveTargetsCommand{
		Source:    "test",
		Timestamp: time.Date(2026, 5, 31, 8, 0, 0, 0, time.UTC),
		Targets: []command.ObserveTargetCommand{
			{
				Channel: command.ChannelCommand{
					Kind:             "qq",
					AccountID:        "1049511700",
					ConversationID:   "27234224",
					ConversationType: "group",
				},
				ObserveOnly: true,
				Enabled:     true,
				Metadata: map[string]string{
					"ragflow_dataset_ids": "ds-hardware, ds-guides, ds-hardware",
				},
			},
			{
				Channel: command.ChannelCommand{
					Kind:             "qq",
					AccountID:        "1049511700",
					ConversationID:   "reply-group",
					ConversationType: "group",
				},
				ObserveOnly: false,
				Enabled:     true,
			},
			{
				Channel: command.ChannelCommand{
					Kind:             "telegram",
					AccountID:        "telegram",
					ConversationID:   "1",
					ConversationType: "group",
				},
				ObserveOnly: true,
				Enabled:     true,
			},
			{
				Channel: command.ChannelCommand{
					Kind:             "qq",
					AccountID:        "1049511700",
					ConversationID:   "disabled",
					ConversationType: "group",
				},
				ObserveOnly: true,
				Enabled:     false,
			},
		},
	})
	if err != nil {
		t.Fatalf("sync observe targets: %v", err)
	}
}

func findPlannedJob(t *testing.T, jobs []query.AgentJobView, jobType string) query.AgentJobView {
	t.Helper()
	for _, job := range jobs {
		if job.JobType == jobType {
			return job
		}
	}
	t.Fatalf("missing planned job type=%s in %#v", jobType, jobs)
	return query.AgentJobView{}
}

func findPlannedDatasetJob(t *testing.T, jobs []query.AgentJobView, datasetID string) query.AgentJobView {
	t.Helper()
	for _, job := range jobs {
		if job.Payload["dataset_id"] == datasetID {
			return job
		}
	}
	t.Fatalf("missing planned dataset job dataset_id=%s in %#v", datasetID, jobs)
	return query.AgentJobView{}
}
