package schedulerleasestore_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/schedulerleasestore"
)

func TestStorePersistsSchedulerExecutionLeases(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scheduler-leases.json")
	store, err := schedulerleasestore.NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	lease := validSchedulerLease("schedule:1")
	if err := store.SaveSchedulerExecutionLease(context.Background(), lease); err != nil {
		t.Fatalf("save lease: %v", err)
	}

	reopened, err := schedulerleasestore.NewStore(path)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	items, err := reopened.ListSchedulerExecutionLeases(context.Background())
	if err != nil {
		t.Fatalf("list leases: %v", err)
	}
	if len(items) != 1 || items[0].JobID != "schedule:1" {
		t.Fatalf("unexpected leases: %+v", items)
	}
}

func TestStoreDeletesSchedulerExecutionLease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scheduler-leases.json")
	store, err := schedulerleasestore.NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	lease := validSchedulerLease("schedule:delete")
	if err := store.SaveSchedulerExecutionLease(context.Background(), lease); err != nil {
		t.Fatalf("save lease: %v", err)
	}
	if err := store.DeleteSchedulerExecutionLease(context.Background(), lease.JobID); err != nil {
		t.Fatalf("delete lease: %v", err)
	}
	items, err := store.ListSchedulerExecutionLeases(context.Background())
	if err != nil {
		t.Fatalf("list leases: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty leases, got %+v", items)
	}
}

func validSchedulerLease(jobID string) model.SchedulerExecutionLease {
	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	lease, err := model.NewSchedulerExecutionLease(model.SchedulerExecutionLeaseSpec{
		JobID:      jobID,
		HolderID:   "scheduler:worker-a",
		LeaseToken: "token-a",
		ExpiresAt:  now.Add(time.Minute),
		UpdatedAt:  now,
	})
	if err != nil {
		panic(err)
	}
	return lease
}
