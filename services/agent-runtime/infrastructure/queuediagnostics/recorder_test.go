package queuediagnostics

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func TestRecorderTracksPublishSuccessAndFailureByKindAndSubject(t *testing.T) {
	inner := &fakeWorkQueuePublisher{agentJobErr: errors.New("nats unavailable")}
	recorder, err := NewRecorder(inner, SubjectResolver{
		OutboxDelivery: func(delivery model.OutboxDelivery) string {
			return "akashic.work.outbox." + delivery.Message.Channel.AccountID
		},
		AgentJob: func(job model.AgentJob) string {
			return "akashic.work.agent_job." + string(job.JobType)
		},
	})
	if err != nil {
		t.Fatalf("new recorder: %v", err)
	}

	if err := recorder.PublishOutboxDelivery(context.Background(), sampleOutboxDelivery(t)); err != nil {
		t.Fatalf("publish outbox: %v", err)
	}
	if err := recorder.PublishAgentJob(context.Background(), sampleAgentJob(t)); err == nil {
		t.Fatalf("expected agent job publish failure")
	}

	snapshot, err := recorder.SnapshotWorkQueuePublishDiagnostics(context.Background())
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if !snapshot.Enabled || snapshot.AttemptTotal != 2 || snapshot.SucceededTotal != 1 || snapshot.FailedTotal != 1 {
		t.Fatalf("unexpected totals: %+v", snapshot)
	}
	if len(snapshot.Subjects) != 2 {
		t.Fatalf("expected subject stats, got %+v", snapshot.Subjects)
	}
	outbox := findWorkKind(t, snapshot.WorkKinds, "outbox_delivery")
	if outbox.SucceededCount != 1 || outbox.FailedCount != 0 || outbox.LastPublishedAt == "" {
		t.Fatalf("unexpected outbox stats: %+v", outbox)
	}
	agentJob := findWorkKind(t, snapshot.WorkKinds, "agent_job")
	if agentJob.SucceededCount != 0 || agentJob.FailedCount != 1 || agentJob.LastError != "nats unavailable" {
		t.Fatalf("unexpected agent job stats: %+v", agentJob)
	}
}

type fakeWorkQueuePublisher struct {
	agentJobErr error
}

func (p *fakeWorkQueuePublisher) PublishOutboxDelivery(_ context.Context, _ model.OutboxDelivery) error {
	return nil
}

func (p *fakeWorkQueuePublisher) PublishAgentJob(_ context.Context, _ model.AgentJob) error {
	return p.agentJobErr
}

func sampleOutboxDelivery(t *testing.T) model.OutboxDelivery {
	t.Helper()
	now := time.Date(2026, 5, 30, 23, 30, 0, 0, time.UTC)
	message := model.OutboundMessage{
		EventID: "outbox:diagnostics:1",
		Channel: model.ChannelRef{
			Kind:             model.ChannelKindQQ,
			AccountID:        "1049511700",
			ConversationID:   "2365524513",
			ConversationType: model.ConversationTypePrivate,
		},
		Content:   "hello",
		Timestamp: now,
	}
	delivery, err := model.NewOutboxDelivery(message, 3, now)
	if err != nil {
		t.Fatalf("new outbox delivery: %v", err)
	}
	return delivery
}

func sampleAgentJob(t *testing.T) model.AgentJob {
	t.Helper()
	now := time.Date(2026, 5, 30, 23, 30, 0, 0, time.UTC)
	job, err := model.NewAgentJob(model.AgentJobSpec{
		JobID:   "agent-job:diagnostics:1",
		JobType: model.AgentJobRagIngest,
		AgentID: "akashic",
		Route: model.ChannelRef{
			Kind:             model.ChannelKindQQ,
			AccountID:        "1049511700",
			ConversationID:   "3219982",
			ConversationType: model.ConversationTypeGroup,
		},
		MaxAttempts: 3,
	}, now)
	if err != nil {
		t.Fatalf("new agent job: %v", err)
	}
	return job
}

func findWorkKind(t *testing.T, items []query.QueuePublishWorkKindStats, workKind string) query.QueuePublishWorkKindStats {
	t.Helper()
	for _, item := range items {
		if item.WorkKind == workKind {
			return item
		}
	}
	t.Fatalf("missing work kind %s in %+v", workKind, items)
	return query.QueuePublishWorkKindStats{}
}
