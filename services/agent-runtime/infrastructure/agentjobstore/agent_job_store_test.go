package agentjobstore_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	store "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/agentjobstore"
)

func TestAgentJobStorePersistsJobsAcrossRestarts(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "agent_jobs.json")
	repo, err := store.NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	now := time.Date(2026, 5, 30, 6, 30, 0, 0, time.UTC)
	jobA, err := model.NewAgentJob(model.AgentJobSpec{
		JobID:   "job-a",
		JobType: model.AgentJobRagIngest,
		AgentID: "worker-test",
		Route: model.ChannelRef{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   "27234224",
			ConversationType: "group",
		},
		SourceEventIDs: []string{"qq:gqq:27234224:1"},
		SourceAssetIDs: []string{"asset:1"},
		Payload:        map[string]string{"k": "v1"},
		MaxAttempts:    2,
		Metadata:       map[string]string{"origin": "python"},
	}, now)
	if err != nil {
		t.Fatalf("new job: %v", err)
	}
	if err := repo.SaveAgentJob(ctx, jobA); err != nil {
		t.Fatalf("save jobA: %v", err)
	}

	jobB, err := model.NewAgentJob(model.AgentJobSpec{
		JobID:   "job-b",
		JobType: model.AgentJobGroupMemoryExtract,
		AgentID: "worker-test",
		Route: model.ChannelRef{
			Kind:             "qq",
			AccountID:        "2365524513",
			ConversationID:   "164369633",
			ConversationType: "group",
		},
		SourceEventIDs: []string{"qq:gqq:164369633:2"},
		SourceAssetIDs: []string{"asset:2"},
		Payload:        map[string]string{"dataset": "group"},
		MaxAttempts:    1,
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("new job: %v", err)
	}
	if err := repo.SaveAgentJob(ctx, jobB); err != nil {
		t.Fatalf("save jobB: %v", err)
	}

	reloaded, err := store.NewStore(path)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}

	stored, ok, err := reloaded.FindAgentJob(ctx, "job-a")
	if err != nil {
		t.Fatalf("find stored job: %v", err)
	}
	if !ok {
		t.Fatalf("expected job-a to persist")
	}
	if stored.Metadata["origin"] != "python" {
		t.Fatalf("expected metadata to persist, got %+v", stored.Metadata)
	}
	items, err := reloaded.ListAgentJobs(ctx, query.AgentJobFilter{Limit: 10})
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(items))
	}
	leaseable, ok, err := reloaded.FindLeaseableAgentJob(ctx, string(model.AgentJobRagIngest), now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("find leaseable: %v", err)
	}
	if !ok {
		t.Fatalf("expected leaseable job")
	}
	if leaseable.JobID != "job-a" {
		t.Fatalf("expected oldest matching job, got %s", leaseable.JobID)
	}
}

func TestAgentJobStoreListAndPersistUpdatedState(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "agent_jobs_update.json")
	repo, err := store.NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	now := time.Date(2026, 5, 30, 7, 0, 0, 0, time.UTC)
	job, err := model.NewAgentJob(model.AgentJobSpec{
		JobID:   "job-updated",
		JobType: model.AgentJobImageGeneration,
		AgentID: "worker-image",
		Route: model.ChannelRef{
			Kind:             "telegram",
			AccountID:        "bot@tg",
			ConversationID:   "chat-1",
			ConversationType: "private",
		},
		SourceEventIDs: []string{"tg:msg:1"},
		MaxAttempts:    3,
	}, now)
	if err != nil {
		t.Fatalf("new job: %v", err)
	}
	if err := job.Lease("old-worker", time.Minute, now); err != nil {
		t.Fatalf("lease job: %v", err)
	}
	if err := job.MarkRunning(now.Add(time.Second)); err != nil {
		t.Fatalf("mark running: %v", err)
	}
	if err := repo.SaveAgentJob(ctx, job); err != nil {
		t.Fatalf("save running job: %v", err)
	}

	reloaded, err := store.NewStore(path)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	stored, ok, err := reloaded.FindAgentJob(ctx, "job-updated")
	if err != nil {
		t.Fatalf("find stored: %v", err)
	}
	if !ok {
		t.Fatalf("expected stored job")
	}
	if stored.Status != model.AgentJobRunning {
		t.Fatalf("expected running status after reload, got %s", stored.Status)
	}
	if stored.LeaseOwner != "old-worker" {
		t.Fatalf("expected lease owner persist, got %s", stored.LeaseOwner)
	}

	pending, ok, err := reloaded.FindLeaseableAgentJob(ctx, string(model.AgentJobImageGeneration), now.Add(10*time.Second))
	if err != nil {
		t.Fatalf("find leaseable: %v", err)
	}
	if ok {
		t.Fatalf("did not expect running job to be leaseable before expiry, got %s", pending.JobID)
	}
}

