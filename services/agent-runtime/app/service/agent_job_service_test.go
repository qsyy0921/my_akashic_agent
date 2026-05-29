package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
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

