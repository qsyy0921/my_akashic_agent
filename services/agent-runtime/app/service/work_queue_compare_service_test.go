package service

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/memory"
)

func TestWorkQueueCompareServiceMatchesLeaseableOutboxState(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 5, 30, 23, 50, 0, 0, time.UTC)
	sender := NewMessageSendServiceWithOutboxEvents(store, store, store, store, store)
	if err := sender.Send(ctx, command.SendMessageCommand{
		EventID: "outbox:compare:1",
		Channel: command.ChannelCommand{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   "2365524513",
			ConversationType: "private",
		},
		Content:   "hello",
		Timestamp: now,
	}); err != nil {
		t.Fatalf("send: %v", err)
	}

	service := NewWorkQueueCompareService(store, store)
	result, err := service.CompareWorkQueueCandidate(ctx, command.CompareWorkQueueCandidateCommand{
		WorkKind:   "outbox_delivery",
		WorkID:     "outbox:compare:1",
		Subject:    "akashic.work.outbox.qq.1049511700",
		ObservedAt: now,
	})
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if !result.Matched || result.Reason != "leaseable_state" || result.StateStatus != "queued" {
		t.Fatalf("expected leaseable outbox match, got %+v", result)
	}

	snapshot, err := service.SnapshotDualReadDiagnostics(ctx)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snapshot.ComparedTotal != 1 || snapshot.MatchedTotal != 1 || snapshot.MismatchedTotal != 0 {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
}

func TestWorkQueueCompareServiceRecordsMissingStateMismatch(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := NewWorkQueueCompareService(store, store)

	result, err := service.CompareWorkQueueCandidate(ctx, command.CompareWorkQueueCandidateCommand{
		WorkKind:   "agent_job",
		WorkID:     "agent-job:missing",
		Subject:    "akashic.work.agent_job.rag_ingest",
		ObservedAt: time.Date(2026, 5, 30, 23, 51, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if result.Matched || result.Reason != "missing_state" {
		t.Fatalf("expected missing state mismatch, got %+v", result)
	}

	snapshot, err := service.SnapshotDualReadDiagnostics(ctx)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snapshot.ComparedTotal != 1 || snapshot.MatchedTotal != 0 || snapshot.MismatchedTotal != 1 {
		t.Fatalf("unexpected snapshot totals: %+v", snapshot)
	}
	if len(snapshot.Reasons) != 1 || snapshot.Reasons[0].Reason != "missing_state" || snapshot.Reasons[0].Count != 1 {
		t.Fatalf("unexpected reasons: %+v", snapshot.Reasons)
	}
}
