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
	if leased.LeaseToken == "" {
		t.Fatalf("expected lease token in lease result: %+v", leased)
	}
	running, err := service.MarkRunning(ctx, command.MarkAgentJobRunningCommand{
		JobID:      "job-svc-1",
		LeaseToken: leased.LeaseToken,
		Timestamp:  now.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("mark running: %v", err)
	}
	if running.Status != string(model.AgentJobRunning) {
		t.Fatalf("expected running, got %s", running.Status)
	}
	done, err := service.Complete(ctx, command.CompleteAgentJobCommand{
		JobID:      "job-svc-1",
		LeaseToken: leased.LeaseToken,
		Result:     map[string]string{"indexed": "true"},
		Timestamp:  now.Add(3 * time.Second),
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if done.Status != string(model.AgentJobSucceeded) || done.Result["indexed"] != "true" {
		t.Fatalf("unexpected completed job: %+v", done)
	}
	if done.LeaseToken != "" || done.LeaseOwner != "" || done.LeaseExpiresAt != "" {
		t.Fatalf("expected completed job to clear active lease: %+v", done)
	}
}

func TestAgentJobServiceRejectsStaleLeaseToken(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := appservice.NewAgentJobService(store)
	now := time.Date(2026, 5, 30, 6, 45, 0, 0, time.UTC)

	if _, err := service.Create(ctx, sampleCreateAgentJobCommand(now)); err != nil {
		t.Fatalf("create job: %v", err)
	}
	first, err := service.LeaseNext(ctx, command.AgentJobLeaseNextCommand{
		WorkerID:   "worker-1",
		JobType:    "rag_ingest",
		TTLSeconds: 60,
		Timestamp:  now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("first lease: %v", err)
	}
	second, err := service.LeaseNext(ctx, command.AgentJobLeaseNextCommand{
		WorkerID:   "worker-2",
		JobType:    "rag_ingest",
		TTLSeconds: 60,
		Timestamp:  now.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("second lease after expiry: %v", err)
	}
	if first.LeaseToken == "" || second.LeaseToken == "" || first.LeaseToken == second.LeaseToken {
		t.Fatalf("expected unique lease tokens: first=%+v second=%+v", first, second)
	}
	if _, err := service.MarkRunning(ctx, command.MarkAgentJobRunningCommand{
		JobID:      "job-svc-1",
		LeaseToken: first.LeaseToken,
		Timestamp:  now.Add(2*time.Minute + time.Second),
	}); err == nil {
		t.Fatal("expected stale lease token to be rejected")
	}
	running, err := service.MarkRunning(ctx, command.MarkAgentJobRunningCommand{
		JobID:      "job-svc-1",
		LeaseToken: second.LeaseToken,
		Timestamp:  now.Add(2*time.Minute + 2*time.Second),
	})
	if err != nil {
		t.Fatalf("mark running with current token: %v", err)
	}
	if running.Status != string(model.AgentJobRunning) {
		t.Fatalf("expected running, got %+v", running)
	}
}

func TestAgentJobServiceStrictLeaseTokenRejectsEmptyResultWriteback(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := appservice.NewAgentJobService(
		store,
		appservice.WithStrictAgentJobLeaseToken(true),
	)
	now := time.Date(2026, 5, 30, 6, 47, 0, 0, time.UTC)

	if _, err := service.Create(ctx, sampleCreateAgentJobCommand(now)); err != nil {
		t.Fatalf("create job: %v", err)
	}
	leased, err := service.LeaseNext(ctx, command.AgentJobLeaseNextCommand{
		WorkerID:   "worker-strict",
		JobType:    "rag_ingest",
		TTLSeconds: 60,
		Timestamp:  now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("lease job: %v", err)
	}
	if _, err := service.MarkRunning(ctx, command.MarkAgentJobRunningCommand{
		JobID:     leased.JobID,
		Timestamp: now.Add(2 * time.Second),
	}); err == nil {
		t.Fatal("expected strict mode to reject empty running token")
	}
	running, err := service.MarkRunning(ctx, command.MarkAgentJobRunningCommand{
		JobID:      leased.JobID,
		LeaseToken: leased.LeaseToken,
		Timestamp:  now.Add(3 * time.Second),
	})
	if err != nil {
		t.Fatalf("mark running with token: %v", err)
	}
	if running.Status != string(model.AgentJobRunning) {
		t.Fatalf("unexpected running job: %+v", running)
	}
	if _, err := service.Complete(ctx, command.CompleteAgentJobCommand{
		JobID:     leased.JobID,
		Result:    map[string]string{"ok": "true"},
		Timestamp: now.Add(4 * time.Second),
	}); err == nil {
		t.Fatal("expected strict mode to reject empty complete token")
	}
	done, err := service.Complete(ctx, command.CompleteAgentJobCommand{
		JobID:      leased.JobID,
		LeaseToken: leased.LeaseToken,
		Result:     map[string]string{"ok": "true"},
		Timestamp:  now.Add(5 * time.Second),
	})
	if err != nil {
		t.Fatalf("complete with token: %v", err)
	}
	if done.Status != string(model.AgentJobSucceeded) {
		t.Fatalf("unexpected completed job: %+v", done)
	}
}

func TestAgentJobServiceLeaseWorkUsesExactQueueWorkID(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := appservice.NewAgentJobService(store)
	now := time.Date(2026, 5, 30, 6, 48, 0, 0, time.UTC)

	if _, err := service.Create(ctx, sampleCreateAgentJobCommand(now)); err != nil {
		t.Fatalf("create job: %v", err)
	}
	leased, err := service.LeaseWork(ctx, command.AgentJobLeaseWorkCommand{
		WorkKind:    "agent_job",
		WorkID:      "job-svc-1",
		AggregateID: "job-svc-1",
		Subject:     "akashic.work.agent_job.rag_ingest",
		WorkerID:    "queue-worker",
		TTLSeconds:  60,
		Timestamp:   now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("lease work: %v", err)
	}
	if leased.JobID != "job-svc-1" || leased.Status != string(model.AgentJobLeased) {
		t.Fatalf("unexpected lease work result: %+v", leased)
	}
	if leased.LeaseOwner != "queue-worker" || leased.LeaseToken == "" {
		t.Fatalf("expected queue worker lease token: %+v", leased)
	}
}

func TestAgentJobServiceLeaseWorkRejectsWrongWorkKindAndAggregateMismatch(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := appservice.NewAgentJobService(store)
	now := time.Date(2026, 5, 30, 6, 49, 0, 0, time.UTC)

	if _, err := service.LeaseWork(ctx, command.AgentJobLeaseWorkCommand{
		WorkKind:   "outbox_delivery",
		WorkID:     "job-svc-1",
		WorkerID:   "queue-worker",
		TTLSeconds: 60,
		Timestamp:  now,
	}); err == nil {
		t.Fatal("expected wrong work kind to be rejected")
	}
	if _, err := service.LeaseWork(ctx, command.AgentJobLeaseWorkCommand{
		WorkKind:    "agent_job",
		WorkID:      "job-svc-1",
		AggregateID: "different-job",
		WorkerID:    "queue-worker",
		TTLSeconds:  60,
		Timestamp:   now,
	}); err == nil {
		t.Fatal("expected aggregate mismatch to be rejected")
	}
}

func TestAgentJobServiceRenewsRunningLeaseWithToken(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := appservice.NewAgentJobService(store)
	now := time.Date(2026, 5, 30, 6, 50, 0, 0, time.UTC)

	if _, err := service.Create(ctx, sampleCreateAgentJobCommand(now)); err != nil {
		t.Fatalf("create job: %v", err)
	}
	leased, err := service.LeaseNext(ctx, command.AgentJobLeaseNextCommand{
		WorkerID:   "worker-renew",
		JobType:    "rag_ingest",
		TTLSeconds: 60,
		Timestamp:  now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("lease job: %v", err)
	}
	if _, err := service.MarkRunning(ctx, command.MarkAgentJobRunningCommand{
		JobID:      leased.JobID,
		LeaseToken: leased.LeaseToken,
		Timestamp:  now.Add(2 * time.Second),
	}); err != nil {
		t.Fatalf("mark running: %v", err)
	}
	renewed, err := service.RenewLease(ctx, command.RenewAgentJobLeaseCommand{
		JobID:      leased.JobID,
		LeaseToken: leased.LeaseToken,
		TTLSeconds: 300,
		Timestamp:  now.Add(30 * time.Second),
	})
	if err != nil {
		t.Fatalf("renew lease: %v", err)
	}
	if renewed.Status != string(model.AgentJobRunning) || renewed.Attempts != leased.Attempts {
		t.Fatalf("renew should preserve running attempt state: %+v", renewed)
	}
	if renewed.LeaseExpiresAt <= leased.LeaseExpiresAt {
		t.Fatalf("expected extended lease expiry, got old=%s new=%s", leased.LeaseExpiresAt, renewed.LeaseExpiresAt)
	}
	if _, err := service.RenewLease(ctx, command.RenewAgentJobLeaseCommand{
		JobID:      leased.JobID,
		LeaseToken: "stale-token",
		TTLSeconds: 300,
		Timestamp:  now.Add(31 * time.Second),
	}); err == nil {
		t.Fatal("expected stale renew token to be rejected")
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
		LeaseToken: "lease-events",
		TTLSeconds: 60,
		Timestamp:  now.Add(time.Second),
	}); err != nil {
		t.Fatalf("lease job: %v", err)
	}
	if _, err := service.MarkRunning(ctx, command.MarkAgentJobRunningCommand{
		JobID:      created.JobID,
		LeaseToken: "lease-events",
		Timestamp:  now.Add(2 * time.Second),
	}); err != nil {
		t.Fatalf("mark running: %v", err)
	}
	if _, err := service.Complete(ctx, command.CompleteAgentJobCommand{
		JobID:      created.JobID,
		LeaseToken: "lease-events",
		Result:     map[string]string{"ok": "true"},
		Timestamp:  now.Add(3 * time.Second),
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
	if _, err := jobs.Create(ctx, sampleKnowledgeJobCommand(
		"rag_eval:qq:284331268:fixture:1",
		"rag_eval",
		"284331268",
		map[string]string{"fixture": "tests/fixtures/group_memory_open_strategy_dataset.json"},
		now.Add(-24*time.Minute),
	)); err != nil {
		t.Fatalf("create rag eval job: %v", err)
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
	if view.Totals["jobs"] != 3 || view.Totals["checkpoints"] != 2 {
		t.Fatalf("unexpected totals: %+v", view.Totals)
	}
	if view.Totals["stale_leases"] != 1 || view.Totals["leaseable_jobs"] != 3 {
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
	ragEval := findKnowledgeWorkerDiagnostic(t, view, "rag_eval")
	if ragEval.StatusCounts["pending"] != 1 || len(ragEval.Checkpoints) != 0 {
		t.Fatalf("unexpected rag eval diagnostic: %+v", ragEval)
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
