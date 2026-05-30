package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/memory"
)

func TestKnowledgeCheckpointServiceUpsertsAndRejectsRegression(t *testing.T) {
	ctx := context.Background()
	service := appservice.NewKnowledgeCheckpointService(memory.NewStore())
	now := time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC)

	created, err := service.Upsert(ctx, command.UpsertKnowledgeCheckpointCommand{
		CheckpointID: "ragflow:qq:27234224:ds1",
		Cursor:       20,
		Metadata:     map[string]string{"group_id": "27234224"},
		Timestamp:    now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Cursor != 20 {
		t.Fatalf("unexpected cursor: %d", created.Cursor)
	}

	found, err := service.Get(ctx, "ragflow:qq:27234224:ds1")
	if err != nil {
		t.Fatal(err)
	}
	if found.Metadata["group_id"] != "27234224" {
		t.Fatalf("metadata not preserved: %+v", found.Metadata)
	}

	_, err = service.Upsert(ctx, command.UpsertKnowledgeCheckpointCommand{
		CheckpointID: "ragflow:qq:27234224:ds1",
		Cursor:       19,
		Timestamp:    now.Add(time.Minute),
	})
	if err == nil {
		t.Fatalf("expected cursor regression to fail")
	}
}

func TestKnowledgeCheckpointServiceListsNewestFirstWithPrefix(t *testing.T) {
	ctx := context.Background()
	service := appservice.NewKnowledgeCheckpointService(memory.NewStore())
	now := time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC)

	for _, item := range []struct {
		id     string
		cursor int
		at     time.Time
	}{
		{"ragflow:qq:27234224:ds1", 10, now},
		{"ragflow:qq:3219982:ds1", 20, now.Add(time.Minute)},
		{"memory:qq:3219982", 30, now.Add(2 * time.Minute)},
	} {
		if _, err := service.Upsert(ctx, command.UpsertKnowledgeCheckpointCommand{
			CheckpointID: item.id,
			Cursor:       item.cursor,
			Timestamp:    item.at,
		}); err != nil {
			t.Fatal(err)
		}
	}

	items, err := service.List(ctx, query.KnowledgeCheckpointFilter{
		Limit:  10,
		Prefix: "ragflow:qq:",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected two ragflow checkpoints, got %d", len(items))
	}
	if items[0].CheckpointID != "ragflow:qq:3219982:ds1" || items[1].CheckpointID != "ragflow:qq:27234224:ds1" {
		t.Fatalf("unexpected ordering: %+v", items)
	}
}
