package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/memory"
)

func TestAgentJobServiceCreateLeaseAndComplete(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := appservice.NewAgentJobService(store)
	now := time.Date(2026, 5, 30, 6, 30, 0, 0, time.UTC)

	created, err := service.Create(ctx, sampleCreateAgentJobCommand(now))
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if created.Status != string(model.AgentJobPending) {
		t.Fatalf("expected pending, got %s", created.Status)
	}
	leased, err := service.LeaseNext(ctx, command.AgentJobLeaseNextCommand{
		WorkerID:   "worker-1",
		JobType:    "rag_ingest",
		TTLSeconds: 60,
		Timestamp:  now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("lease next: %v", err)
	}
	if leased.JobID != "job-svc-1" || leased.Status != string(model.AgentJobLeased) {
		t.Fatalf("unexpected lease result: %+v", leased)
	}
	running, err := service.MarkRunning(ctx, command.MarkAgentJobRunningCommand{
		JobID:     "job-svc-1",
		Timestamp: now.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("mark running: %v", err)
	}
	if running.Status != string(model.AgentJobRunning) {
		t.Fatalf("expected running, got %s", running.Status)
	}
	done, err := service.Complete(ctx, command.CompleteAgentJobCommand{
		JobID:     "job-svc-1",
		Result:    map[string]string{"indexed": "true"},
		Timestamp: now.Add(3 * time.Second),
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if done.Status != string(model.AgentJobSucceeded) || done.Result["indexed"] != "true" {
		t.Fatalf("unexpected completed job: %+v", done)
	}
}

func TestAgentJobServiceDuplicateCreateIsIdempotent(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := appservice.NewAgentJobService(store)
	now := time.Date(2026, 5, 30, 6, 30, 0, 0, time.UTC)

	cmd := sampleCreateAgentJobCommand(now)
	first, err := service.Create(ctx, cmd)
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	second, err := service.Create(ctx, cmd)
	if err != nil {
		t.Fatalf("create second: %v", err)
	}
	if second.JobID != first.JobID {
		t.Fatalf("expected idempotent job id, got %s and %s", first.JobID, second.JobID)
	}
	if len(store.AgentJobs()) != 1 {
		t.Fatalf("expected one job, got %d", len(store.AgentJobs()))
	}
}

func TestAgentJobServiceWritesLifecycleEvents(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := appservice.NewAgentJobServiceWithEvents(store, store)
	events := appservice.NewAgentJobEventService(store)
	now := time.Date(2026, 5, 30, 7, 30, 0, 0, time.UTC)

	created, err := service.Create(ctx, sampleCreateAgentJobCommand(now))
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if _, err := service.Lease(ctx, command.AgentJobLeaseCommand{
		JobID:      created.JobID,
		WorkerID:   "worker-events",
		TTLSeconds: 60,
		Timestamp:  now.Add(time.Second),
	}); err != nil {
		t.Fatalf("lease job: %v", err)
	}
	if _, err := service.MarkRunning(ctx, command.MarkAgentJobRunningCommand{
		JobID:     created.JobID,
		Timestamp: now.Add(2 * time.Second),
	}); err != nil {
		t.Fatalf("mark running: %v", err)
	}
	if _, err := service.Complete(ctx, command.CompleteAgentJobCommand{
		JobID:     created.JobID,
		Result:    map[string]string{"ok": "true"},
		Timestamp: now.Add(3 * time.Second),
	}); err != nil {
		t.Fatalf("complete: %v", err)
	}

	items, err := events.List(ctx, query.AgentJobEventFilter{
		JobID: created.JobID,
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if got := len(items); got != 4 {
		t.Fatalf("expected 4 events, got %d: %+v", got, items)
	}
	if items[0].EventType != "succeeded" || items[1].EventType != "running" || items[2].EventType != "leased" || items[3].EventType != "created" {
		t.Fatalf("unexpected event order: %+v", items)
	}
	if items[0].Status != "succeeded" || items[2].LeaseOwner != "worker-events" {
		t.Fatalf("unexpected event payloads: %+v", items)
	}
}

func TestKnowledgeWorkerDiagnosticsServiceSummarizesMemoryAndRAGJobs(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	jobs := appservice.NewAgentJobService(store)
	checkpoints := appservice.NewKnowledgeCheckpointService(store)
	diagnostics := appservice.NewKnowledgeWorkerDiagnosticsService(store, store)
	now := time.Date(2026, 5, 30, 8, 0, 0, 0, time.UTC)

	if _, err := jobs.Create(ctx, sampleKnowledgeJobCommand(
		"group_memory_extract:qq:27234224:1",
		"group_memory_extract",
		"27234224",
		map[string]string{"group_id": "27234224"},
		now.Add(-30*time.Minute),
	)); err != nil {
		t.Fatalf("create group memory job: %v", err)
	}
	if _, err := jobs.Create(ctx, sampleKnowledgeJobCommand(
		"rag_ingest:qq:3219982:ds1:1",
		"rag_ingest",
		"3219982",
		map[string]string{"group_id": "3219982", "dataset_id": "ds1"},
		now.Add(-25*time.Minute),
	)); err != nil {
		t.Fatalf("create rag job: %v", err)
	}
	if _, err := jobs.LeaseNext(ctx, command.AgentJobLeaseNextCommand{
		WorkerID:   "knowledge-worker",
		JobType:    "rag_ingest",
		TTLSeconds: 60,
		Timestamp:  now.Add(-20 * time.Minute),
	}); err != nil {
		t.Fatalf("lease rag job: %v", err)
	}
	if _, err := checkpoints.Upsert(ctx, command.UpsertKnowledgeCheckpointCommand{
		CheckpointID: "ragflow:qq:3219982:ds1",
		Cursor:       128,
		Metadata:     map[string]string{"group_id": "3219982", "dataset_id": "ds1"},
		Timestamp:    now.Add(-10 * time.Minute),
	}); err != nil {
		t.Fatalf("upsert rag checkpoint: %v", err)
	}
	if _, err := checkpoints.Upsert(ctx, command.UpsertKnowledgeCheckpointCommand{
		CheckpointID: "memory:qq:27234224",
		Cursor:       512,
		Metadata:     map[string]string{"group_id": "27234224"},
		Timestamp:    now.Add(-5 * time.Minute),
	}); err != nil {
		t.Fatalf("upsert memory checkpoint: %v", err)
	}

	view, err := diagnostics.Get(ctx, query.KnowledgeWorkerDiagnosticsFilter{
		Limit:             10,
		StaleAfterSeconds: 300,
		Now:               now,
	})
	if err != nil {
		t.Fatalf("diagnostics: %v", err)
	}
	if view.Totals["jobs"] != 2 || view.Totals["checkpoints"] != 2 {
		t.Fatalf("unexpected totals: %+v", view.Totals)
	}
	if view.Totals["stale_leases"] != 1 || view.Totals["leaseable_jobs"] != 2 {
		t.Fatalf("unexpected operational totals: %+v", view.Totals)
	}
	group := findKnowledgeWorkerDiagnostic(t, view, "group_memory_extract")
	if group.StatusCounts["pending"] != 1 || len(group.Checkpoints) != 1 {
		t.Fatalf("unexpected group diagnostic: %+v", group)
	}
	if group.LatestCheckpoint == nil || group.LatestCheckpoint.CheckpointID != "memory:qq:27234224" {
		t.Fatalf("unexpected group checkpoint: %+v", group.LatestCheckpoint)
	}
	rag := findKnowledgeWorkerDiagnostic(t, view, "rag_ingest")
	if rag.StatusCounts["leased"] != 1 || rag.StaleLeaseCount != 1 {
		t.Fatalf("unexpected rag diagnostic: %+v", rag)
	}
	if rag.LatestCheckpoint == nil || rag.LatestCheckpoint.Cursor != 128 {
		t.Fatalf("unexpected rag checkpoint: %+v", rag.LatestCheckpoint)
	}
}

func sampleCreateAgentJobCommand(timestamp time.Time) command.CreateAgentJobCommand {
	return command.CreateAgentJobCommand{
		JobID:   "job-svc-1",
		JobType: "rag_ingest",
		AgentID: "main",
		Route: command.ChannelCommand{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   "27234224",
			ConversationType: "group",
		},
		SourceEventIDs: []string{"qq:gqq:27234224:1"},
		SourceAssetIDs: []string{"asset:1"},
		Payload:        map[string]string{"dataset": "group"},
		MaxAttempts:    2,
		Timestamp:      timestamp,
	}
}

func sampleKnowledgeJobCommand(
	jobID string,
	jobType string,
	groupID string,
	payload map[string]string,
	timestamp time.Time,
) command.CreateAgentJobCommand {
	return command.CreateAgentJobCommand{
		JobID:   jobID,
		JobType: jobType,
		AgentID: "knowledge-worker",
		Route: command.ChannelCommand{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   groupID,
			ConversationType: "group",
		},
		Payload:     payload,
		MaxAttempts: 2,
		Timestamp:   timestamp,
	}
}

func findKnowledgeWorkerDiagnostic(
	t *testing.T,
	view query.KnowledgeWorkerDiagnosticsView,
	jobType string,
) query.KnowledgeWorkerDiagnosticView {
	t.Helper()
	for _, worker := range view.Workers {
		if worker.JobType == jobType {
			return worker
		}
	}
	t.Fatalf("worker diagnostic not found for %s: %+v", jobType, view.Workers)
	return query.KnowledgeWorkerDiagnosticView{}
}
