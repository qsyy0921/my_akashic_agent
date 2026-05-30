package knowledgecheckpointstore_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/knowledgecheckpointstore"
)

func TestKnowledgeCheckpointStorePersistsCheckpoint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "knowledge-checkpoints.json")
	store, err := knowledgecheckpointstore.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	checkpoint, err := model.NewKnowledgeCheckpoint(
		"ragflow:qq:27234224:ds1",
		42,
		map[string]string{"group_id": "27234224"},
		time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveKnowledgeCheckpoint(context.Background(), checkpoint); err != nil {
		t.Fatal(err)
	}

	reopened, err := knowledgecheckpointstore.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	found, ok, err := reopened.FindKnowledgeCheckpoint(context.Background(), "ragflow:qq:27234224:ds1")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || found.Cursor != 42 || found.Metadata["group_id"] != "27234224" {
		t.Fatalf("unexpected persisted checkpoint: ok=%v checkpoint=%+v", ok, found)
	}
}
