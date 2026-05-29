package model_test

import (
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/domain/model"
)

func TestAgentJobLeaseFailureRetryAndDeadLetter(t *testing.T) {
	now := time.Date(2026, 5, 30, 6, 0, 0, 0, time.UTC)
	job, err := model.NewAgentJob(sampleAgentJobSpec(2), now)
	if err != nil {
		t.Fatalf("new job: %v", err)
	}
	if err := job.Lease("worker-1", time.Minute, now.Add(time.Second)); err != nil {
		t.Fatalf("lease: %v", err)
	}
	if job.Status != model.AgentJobLeased || job.Attempts != 1 {
		t.Fatalf("unexpected lease state: %+v", job)
	}
	if err := job.MarkRunning(now.Add(2 * time.Second)); err != nil {
		t.Fatalf("running: %v", err)
	}
	if err := job.MarkFailed("timeout", now.Add(3*time.Second)); err != nil {
		t.Fatalf("failed: %v", err)
	}
	if job.Status != model.AgentJobFailed {
		t.Fatalf("expected failed, got %s", job.Status)
	}
	if err := job.Retry(now.Add(4 * time.Second)); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if err := job.Lease("worker-1", time.Minute, now.Add(5*time.Second)); err != nil {
		t.Fatalf("second lease: %v", err)
	}
	if err := job.MarkFailed("again", now.Add(6*time.Second)); err != nil {
		t.Fatalf("second fail: %v", err)
	}
	if job.Status != model.AgentJobDeadLettered {
		t.Fatalf("expected dead letter, got %s", job.Status)
	}
}

func TestAgentJobExpiredLeaseCanBeLeasedAgain(t *testing.T) {
	now := time.Date(2026, 5, 30, 6, 0, 0, 0, time.UTC)
	job, err := model.NewAgentJob(sampleAgentJobSpec(3), now)
	if err != nil {
		t.Fatalf("new job: %v", err)
	}
	if err := job.Lease("worker-1", time.Minute, now); err != nil {
		t.Fatalf("lease: %v", err)
	}
	if !job.CanLease(now.Add(2 * time.Minute)) {
		t.Fatal("expected expired lease to be leaseable")
	}
	if err := job.Lease("worker-2", time.Minute, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("re-lease expired job: %v", err)
	}
	if job.LeaseOwner != "worker-2" {
		t.Fatalf("expected worker-2 lease owner, got %s", job.LeaseOwner)
	}
}

func TestCancelledAgentJobCannotBeLeased(t *testing.T) {
	now := time.Date(2026, 5, 30, 6, 0, 0, 0, time.UTC)
	job, err := model.NewAgentJob(sampleAgentJobSpec(3), now)
	if err != nil {
		t.Fatalf("new job: %v", err)
	}
	if err := job.Cancel(now.Add(time.Second)); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if err := job.Lease("worker", time.Minute, now.Add(2*time.Second)); err == nil {
		t.Fatal("expected cancelled job lease to fail")
	}
}

func sampleAgentJobSpec(maxAttempts int) model.AgentJobSpec {
	return model.AgentJobSpec{
		JobID:   "job-1",
		JobType: model.AgentJobRagIngest,
		AgentID: "main",
		Route: model.ChannelRef{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   "27234224",
			ConversationType: "group",
		},
		SourceEventIDs: []string{"qq:gqq:27234224:1"},
		SourceAssetIDs: []string{"asset:1"},
		MaxAttempts:    maxAttempts,
	}
}
