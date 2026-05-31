package model

import (
	"testing"
	"time"
)

func TestSchedulerExecutionLeaseActiveMatchesAndRenews(t *testing.T) {
	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	lease, err := NewSchedulerExecutionLease(SchedulerExecutionLeaseSpec{
		JobID:      "schedule:1",
		HolderID:   "scheduler:worker-a",
		LeaseToken: "tok-1",
		ExpiresAt:  now.Add(time.Minute),
		UpdatedAt:  now,
	})
	if err != nil {
		t.Fatalf("new scheduler execution lease: %v", err)
	}
	if !lease.ActiveAt(now) || lease.ActiveAt(now.Add(2*time.Minute)) {
		t.Fatalf("unexpected active state: %+v", lease)
	}
	if !lease.Matches("scheduler:worker-a", "tok-1") || lease.Matches("scheduler:worker-b", "tok-1") {
		t.Fatalf("unexpected match state: %+v", lease)
	}

	renewed, err := lease.Renew(now.Add(5*time.Minute), now.Add(30*time.Second))
	if err != nil {
		t.Fatalf("renew scheduler execution lease: %v", err)
	}
	if !renewed.ExpiresAt.Equal(now.Add(5 * time.Minute)) {
		t.Fatalf("unexpected renewed expiry: %s", renewed.ExpiresAt)
	}
}

func TestSchedulerExecutionLeaseRequiresToken(t *testing.T) {
	_, err := NewSchedulerExecutionLease(SchedulerExecutionLeaseSpec{
		JobID:     "schedule:1",
		HolderID:  "scheduler:worker-a",
		ExpiresAt: time.Now().UTC().Add(time.Minute),
		UpdatedAt: time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("expected missing token to fail")
	}
}
