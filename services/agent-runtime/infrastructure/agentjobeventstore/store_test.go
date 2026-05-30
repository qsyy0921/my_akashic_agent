package agentjobeventstore_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	store "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/agentjobeventstore"
)

func TestAgentJobEventStoreAppendsAndListsEvents(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "agent-job-events.jsonl")
	events, err := store.NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	now := time.Date(2026, 5, 30, 9, 0, 0, 0, time.UTC)
	for _, item := range []struct {
		id        string
		jobID     string
		eventType model.AgentJobEventType
		at        time.Time
	}{
		{"event-1", "job-1", model.AgentJobEventCreated, now},
		{"event-2", "job-1", model.AgentJobEventLeased, now.Add(time.Second)},
		{"event-3", "job-2", model.AgentJobEventCreated, now.Add(2 * time.Second)},
	} {
		event, err := model.NewAgentJobEvent(model.AgentJobEventSpec{
			EventID:     item.id,
			JobID:       item.jobID,
			JobType:     model.AgentJobRagIngest,
			EventType:   item.eventType,
			Status:      model.AgentJobPending,
			Attempt:     0,
			MaxAttempts: 2,
		}, item.at)
		if err != nil {
			t.Fatalf("new event: %v", err)
		}
		if err := events.AppendAgentJobEvent(ctx, event); err != nil {
			t.Fatalf("append event: %v", err)
		}
	}

	reloaded, err := store.NewStore(path)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	items, err := reloaded.ListAgentJobEvents(ctx, query.AgentJobEventFilter{
		JobID: "job-1",
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 events, got %d", len(items))
	}
	if items[0].EventID != "event-2" || items[1].EventID != "event-1" {
		t.Fatalf("unexpected order: %+v", items)
	}
}
