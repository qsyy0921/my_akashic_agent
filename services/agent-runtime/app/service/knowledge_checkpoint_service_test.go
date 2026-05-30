package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
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
