package service

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/memory"
)

func TestQueueBackendServiceReconcilesShadowPublishAgainstStateAndEvents(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 5, 30, 23, 45, 0, 0, time.UTC)

	sender := NewMessageSendServiceWithOutboxEventsAndWorkQueue(store, store, store, store, store, nil)
	if err := sender.Send(ctx, command.SendMessageCommand{
		EventID: "outbox:diagnostics:1",
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

	jobs := NewAgentJobServiceWithEvents(store, store)
	if _, err := jobs.Create(ctx, command.CreateAgentJobCommand{
		JobID:   "agent-job:diagnostics:1",
		JobType: "rag_ingest",
		AgentID: "akashic",
		Route: command.ChannelCommand{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   "3219982",
			ConversationType: "group",
		},
		MaxAttempts: 3,
		Timestamp:   now,
	}); err != nil {
		t.Fatalf("create job: %v", err)
	}

	service := NewQueueBackendServiceWithDiagnostics(query.QueueBackendView{
		Provider:       "nats_jetstream",
		Mode:           "shadow_publish",
		MigrationPhase: "shadow_ready",
	}, QueueBackendDiagnosticsDeps{
		Diagnostics: fakeQueueDiagnosticsReader{
			snapshot: query.QueueShadowPublishDiagnostics{
				Enabled:        true,
				AttemptTotal:   2,
				SucceededTotal: 1,
				FailedTotal:    1,
				WorkKinds: []query.QueuePublishWorkKindStats{
					{WorkKind: "outbox_delivery", AttemptCount: 1, SucceededCount: 1},
					{WorkKind: "agent_job", AttemptCount: 1, FailedCount: 1, LastError: "nats unavailable"},
				},
				Subjects: []query.QueuePublishSubjectStats{
					{Subject: "akashic.work.outbox.qq.1049511700", WorkKind: "outbox_delivery", AttemptCount: 1, SucceededCount: 1},
				},
			},
		},
		OutboxRepo:     store,
		OutboxEvents:   store,
		AgentJobRepo:   store,
		AgentJobEvents: store,
	})

	view, err := service.Get(ctx)
	if err != nil {
		t.Fatalf("get queue backend: %v", err)
	}
	if view.ShadowPublish == nil {
		t.Fatalf("expected shadow publish diagnostics")
	}
	diagnostics := view.ShadowPublish
	if !diagnostics.Enabled || diagnostics.StateStoreTotal != 2 || diagnostics.EventStreamTotal != 2 {
		t.Fatalf("unexpected reconciliation totals: %+v", diagnostics)
	}
	if diagnostics.StateMinusPublishedTotal != 1 || diagnostics.EventMinusPublishedTotal != 1 {
		t.Fatalf("unexpected publish deltas: %+v", diagnostics)
	}
	outbox := findQueueWorkKind(t, diagnostics.WorkKinds, "outbox_delivery")
	if outbox.StateStoreCount != 1 || outbox.EventStreamCount != 1 || outbox.StateMinusPublished != 0 {
		t.Fatalf("unexpected outbox reconciliation: %+v", outbox)
	}
	agentJob := findQueueWorkKind(t, diagnostics.WorkKinds, "agent_job")
	if agentJob.StateStoreCount != 1 || agentJob.EventStreamCount != 1 || agentJob.StateMinusPublished != 1 || agentJob.LastError != "nats unavailable" {
		t.Fatalf("unexpected agent job reconciliation: %+v", agentJob)
	}
}

type fakeQueueDiagnosticsReader struct {
	snapshot query.QueueShadowPublishDiagnostics
}

func (r fakeQueueDiagnosticsReader) SnapshotWorkQueuePublishDiagnostics(context.Context) (query.QueueShadowPublishDiagnostics, error) {
	return r.snapshot, nil
}

func findQueueWorkKind(t *testing.T, items []query.QueuePublishWorkKindStats, workKind string) query.QueuePublishWorkKindStats {
	t.Helper()
	for _, item := range items {
		if item.WorkKind == workKind {
			return item
		}
	}
	t.Fatalf("missing work kind %s in %+v", workKind, items)
	return query.QueuePublishWorkKindStats{}
}
