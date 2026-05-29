package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/app/command"
	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-gateway/app/service"
	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/infrastructure/memory"
)

func TestImageJobServiceCreatesIdempotentQueuedJob(t *testing.T) {
	store := memory.NewStore()
	service := appservice.NewImageJobService(store, store)
	cmd := command.CreateImageJobCommand{
		RequestID: "qq_2365524513:1049511700:1",
		Requester: command.ChannelCommand{
			Kind:             "qq",
			AccountID:        "2365524513",
			ConversationID:   "1049511700",
			ConversationType: "private",
		},
		RequesterID: "1049511700",
		Prompt:      "生成一张古装人物图",
		Provider:    "chatgpt-web",
		Model:       "gpt-image",
		Size:        "1024x1024",
		Count:       1,
		Timestamp:   time.Date(2026, 5, 30, 1, 0, 0, 0, time.UTC),
	}

	first, err := service.Create(context.Background(), cmd)
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}
	second, err := service.Create(context.Background(), cmd)
	if err != nil {
		t.Fatalf("second create returned error: %v", err)
	}
	if first.JobID != second.JobID {
		t.Fatalf("expected idempotent job id, got %s and %s", first.JobID, second.JobID)
	}
	if len(store.ImageQueue()) != 1 {
		t.Fatalf("expected exactly one queued job, got %d", len(store.ImageQueue()))
	}
}

func TestImageJobServiceCreatesGenericAgentJobCompatibilityRecord(t *testing.T) {
	store := memory.NewStore()
	service := appservice.NewImageJobServiceWithAgentJobs(store, store, store)
	job, err := service.Create(context.Background(), command.CreateImageJobCommand{
		RequestID: "qq_2365524513:1049511700:2",
		Requester: command.ChannelCommand{
			Kind:             "qq",
			AccountID:        "2365524513",
			ConversationID:   "1049511700",
			ConversationType: "private",
		},
		RequesterID: "1049511700",
		Prompt:      "生成一张古装人物图",
		Provider:    "chatgpt-web",
		Model:       "gpt-image",
		Size:        "1024x1024",
		Count:       2,
		MaxAttempts: 4,
		Timestamp:   time.Date(2026, 5, 30, 1, 0, 0, 0, time.UTC),
		Metadata: map[string]string{
			"source_event_id": "qq:event:2",
			"worker_agent_id": "image-worker",
		},
	})
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	generic, ok, err := store.FindAgentJob(context.Background(), job.JobID)
	if err != nil {
		t.Fatalf("find generic job returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected generic image_generation agent job")
	}
	if generic.JobType != model.AgentJobImageGeneration {
		t.Fatalf("unexpected generic job type: %s", generic.JobType)
	}
	if generic.AgentID != "image-worker" {
		t.Fatalf("unexpected agent id: %s", generic.AgentID)
	}
	if generic.Route.AccountID != "2365524513" || generic.Route.ConversationID != "1049511700" {
		t.Fatalf("unexpected route: %+v", generic.Route)
	}
	if generic.Payload["prompt"] != "生成一张古装人物图" || generic.Payload["count"] != "2" {
		t.Fatalf("unexpected payload: %+v", generic.Payload)
	}
	if len(generic.SourceEventIDs) != 1 || generic.SourceEventIDs[0] != "qq:event:2" {
		t.Fatalf("unexpected source events: %+v", generic.SourceEventIDs)
	}
	if generic.MaxAttempts != 4 {
		t.Fatalf("unexpected max attempts: %d", generic.MaxAttempts)
	}
}

func TestImageJobServiceCompletesJobWithResultAttachment(t *testing.T) {
	store := memory.NewStore()
	service := appservice.NewImageJobService(store, store)
	job, err := service.Create(context.Background(), command.CreateImageJobCommand{
		RequestID: "request-2",
		Requester: command.ChannelCommand{
			Kind:             "telegram",
			AccountID:        "dongri0909bot",
			ConversationID:   "123",
			ConversationType: "private",
		},
		Prompt:    "生成机械键盘图片",
		Timestamp: time.Date(2026, 5, 30, 1, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	running, err := service.MarkRunning(context.Background(), command.MarkImageJobRunningCommand{JobID: job.JobID})
	if err != nil {
		t.Fatalf("mark running returned error: %v", err)
	}
	if running.Status != string(model.ImageJobStatusRunning) || running.Attempts != 1 {
		t.Fatalf("unexpected running job: status=%s attempts=%d", running.Status, running.Attempts)
	}

	completed, err := service.Complete(context.Background(), command.CompleteImageJobCommand{
		JobID: job.JobID,
		Results: []command.AttachmentCommand{{
			Kind:     "image",
			URL:      "file:///workspace/generated.png",
			MimeType: "image/png",
			Name:     "generated.png",
		}},
	})
	if err != nil {
		t.Fatalf("complete returned error: %v", err)
	}
	if completed.Status != string(model.ImageJobStatusSucceeded) || len(completed.Results) != 1 {
		t.Fatalf("unexpected completed job: status=%s results=%d", completed.Status, len(completed.Results))
	}
}
