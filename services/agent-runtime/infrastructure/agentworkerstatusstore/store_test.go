package agentworkerstatusstore

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func TestStorePersistsAgentWorkerStatuses(t *testing.T) {
	path := t.TempDir() + "/agent-worker-statuses.json"
	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	status, err := model.NewAgentWorkerStatus(model.AgentWorkerStatusSpec{
		WorkerID:       "worker-a",
		InstanceID:     "instance-a",
		WorkerType:     "image_generation",
		Status:         "idle",
		ProcessedTotal: 3,
		Source:         "python",
		LeaseUntil:     time.Date(2026, 5, 31, 10, 2, 0, 0, time.UTC),
	}, time.Date(2026, 5, 31, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new status: %v", err)
	}
	if err := store.SaveAgentWorkerStatus(context.Background(), status); err != nil {
		t.Fatalf("save status: %v", err)
	}

	reopened, err := NewStore(path)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	items, err := reopened.ListAgentWorkerStatuses(context.Background())
	if err != nil {
		t.Fatalf("list statuses: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].WorkerID != "worker-a" ||
		items[0].InstanceID != "instance-a" ||
		items[0].ProcessedTotal != 3 ||
		items[0].LeaseUntil.IsZero() {
		t.Fatalf("unexpected persisted item: %#v", items[0])
	}
}
