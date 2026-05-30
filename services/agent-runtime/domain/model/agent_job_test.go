package model_test

import (
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func TestAgentJobLeaseFailureRetryAndDeadLetter(t *testing.T) {
	now := time.Date(2026, 5, 30, 6, 0, 0, 0, time.UTC)
	job, err := model.NewAgentJob(sampleAgentJobSpec(2), now)
	if err != nil {
		t.Fatalf("new job: %v", err)
	}
	if err := job.Lease("worker-1", time.Minute, "lease-1", now.Add(time.Second)); err != nil {
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
	if err := job.Lease("worker-1", time.Minute, "lease-2", now.Add(5*time.Second)); err != nil {
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
	if err := job.Lease("worker-1", time.Minute, "lease-1", now); err != nil {
		t.Fatalf("lease: %v", err)
	}
	if !job.CanLease(now.Add(2 * time.Minute)) {
		t.Fatal("expected expired lease to be leaseable")
	}
	if err := job.Lease("worker-2", time.Minute, "lease-2", now.Add(2*time.Minute)); err != nil {
		t.Fatalf("re-lease expired job: %v", err)
	}
	if job.LeaseOwner != "worker-2" {
		t.Fatalf("expected worker-2 lease owner, got %s", job.LeaseOwner)
	}
}

func TestAgentJobRenewLeaseExtendsRunningLease(t *testing.T) {
	now := time.Date(2026, 5, 30, 6, 10, 0, 0, time.UTC)
	job, err := model.NewAgentJob(sampleAgentJobSpec(3), now)
	if err != nil {
		t.Fatalf("new job: %v", err)
	}
	if err := job.Lease("worker-1", time.Minute, "lease-1", now); err != nil {
		t.Fatalf("lease: %v", err)
	}
	if err := job.MarkRunning(now.Add(10 * time.Second)); err != nil {
		t.Fatalf("running: %v", err)
	}
	if err := job.RenewLease("lease-1", 5*time.Minute, now.Add(30*time.Second)); err != nil {
		t.Fatalf("renew: %v", err)
	}
	if job.Status != model.AgentJobRunning || job.Attempts != 1 {
		t.Fatalf("renew should preserve running attempt state: %+v", job)
	}
	if want := now.Add(30*time.Second + 5*time.Minute); !job.LeaseExpiresAt.Equal(want) {
		t.Fatalf("unexpected renewed expiry: got %s want %s", job.LeaseExpiresAt, want)
	}
	if err := job.RenewLease("stale", time.Minute, now.Add(40*time.Second)); err == nil {
		t.Fatal("expected stale renew token to fail")
	}
}

func TestAgentJobRenewExpiredLeaseFails(t *testing.T) {
	now := time.Date(2026, 5, 30, 6, 20, 0, 0, time.UTC)
	job, err := model.NewAgentJob(sampleAgentJobSpec(3), now)
	if err != nil {
		t.Fatalf("new job: %v", err)
	}
	if err := job.Lease("worker-1", time.Minute, "lease-1", now); err != nil {
		t.Fatalf("lease: %v", err)
	}
	if err := job.RenewLease("lease-1", time.Minute, now.Add(2*time.Minute)); err == nil {
		t.Fatal("expected expired lease renew to fail")
	}
}

func TestAgentJobRecoverExpiredLeaseReturnsPendingOrDeadLetter(t *testing.T) {
	now := time.Date(2026, 5, 30, 6, 25, 0, 0, time.UTC)
	job, err := model.NewAgentJob(sampleAgentJobSpec(2), now)
	if err != nil {
		t.Fatalf("new job: %v", err)
	}
	if err := job.Lease("worker-1", time.Minute, "lease-1", now); err != nil {
		t.Fatalf("lease: %v", err)
	}
	action, err := job.RecoverExpiredLease(now.Add(2 * time.Minute))
	if err != nil {
		t.Fatalf("recover expired lease: %v", err)
	}
	if action != model.AgentJobLeaseRecovered || job.Status != model.AgentJobPending {
		t.Fatalf("expected recovered pending job, action=%s job=%+v", action, job)
	}
	if job.LeaseOwner != "" || job.LeaseToken != "" || !job.LeaseExpiresAt.IsZero() {
		t.Fatalf("expected recovery to clear active lease: %+v", job)
	}

	if err := job.Lease("worker-2", time.Minute, "lease-2", now.Add(3*time.Minute)); err != nil {
		t.Fatalf("second lease: %v", err)
	}
	action, err = job.RecoverExpiredLease(now.Add(5 * time.Minute))
	if err != nil {
		t.Fatalf("recover exhausted lease: %v", err)
	}
	if action != model.AgentJobLeaseDeadLettered || job.Status != model.AgentJobDeadLettered {
		t.Fatalf("expected exhausted job to dead-letter, action=%s job=%+v", action, job)
	}
	if job.ErrorMessage == "" {
		t.Fatalf("expected dead-letter reason: %+v", job)
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
	if err := job.Lease("worker", time.Minute, "lease-1", now.Add(2*time.Second)); err == nil {
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
