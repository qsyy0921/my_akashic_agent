package model_test

import (
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func TestKnowledgeCheckpointAdvanceRejectsBackwardsCursor(t *testing.T) {
	now := time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC)
	checkpoint, err := model.NewKnowledgeCheckpoint("ragflow:qq:27234224:ds1", 10, map[string]string{"kind": "ragflow"}, now)
	if err != nil {
		t.Fatal(err)
	}

	if err := checkpoint.Advance(9, nil, now.Add(time.Minute)); err == nil {
		t.Fatalf("expected backwards cursor to fail")
	}
	if err := checkpoint.Advance(11, map[string]string{"kind": "ragflow", "group_id": "27234224"}, now.Add(time.Minute)); err != nil {
		t.Fatalf("advance failed: %v", err)
	}
	if checkpoint.Cursor != 11 || checkpoint.Metadata["group_id"] != "27234224" {
		t.Fatalf("unexpected checkpoint after advance: %+v", checkpoint)
	}
}
