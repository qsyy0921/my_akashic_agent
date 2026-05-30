package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/memory"
)

func TestMessageSendServiceShadowPublishesOutboxWork(t *testing.T) {
	store := memory.NewStore()
	publisher := &recordingWorkQueuePublisher{}
	sender := appservice.NewMessageSendServiceWithOutboxEventsAndWorkQueue(store, store, store, store, store, publisher)

	err := sender.Send(context.Background(), command.SendMessageCommand{
		EventID: "outbox-shadow-1",
		Channel: command.ChannelCommand{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   "2365524513",
			ConversationType: "private",
		},
		Content:   "shadow publish",
		Timestamp: time.Date(2026, 5, 30, 22, 30, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("send returned error: %v", err)
	}

	if len(publisher.outbox) != 1 || publisher.outbox[0].Message.EventID != "outbox-shadow-1" {
		t.Fatalf("expected outbox work notification, got %+v", publisher.outbox)
	}
}

func TestMessageSendServiceIgnoresShadowPublishFailureAfterStateCommit(t *testing.T) {
	store := memory.NewStore()
	publisher := &recordingWorkQueuePublisher{err: errors.New("nats unavailable")}
	sender := appservice.NewMessageSendServiceWithOutboxEventsAndWorkQueue(store, store, store, store, store, publisher)

	err := sender.Send(context.Background(), command.SendMessageCommand{
		EventID: "outbox-shadow-fail",
		Channel: command.ChannelCommand{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   "2365524513",
			ConversationType: "private",
		},
		Content:   "shadow publish failure",
		Timestamp: time.Date(2026, 5, 30, 22, 31, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("shadow publish failure must not fail send: %v", err)
	}
	if len(store.OutboxDeliveries()) != 1 {
		t.Fatalf("expected committed outbox state, got %d", len(store.OutboxDeliveries()))
	}
	if len(store.Outbound()) != 1 {
		t.Fatalf("expected outbound event after shadow failure, got %d", len(store.Outbound()))
	}
}

func TestAgentJobServiceShadowPublishesCreatedJobWork(t *testing.T) {
	store := memory.NewStore()
	publisher := &recordingWorkQueuePublisher{}
	service := appservice.NewAgentJobServiceWithEventsAndWorkQueue(store, store, publisher)
	now := time.Date(2026, 5, 30, 22, 32, 0, 0, time.UTC)

	job, err := service.Create(context.Background(), sampleCreateAgentJobCommand(now))
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	if len(publisher.jobs) != 1 || publisher.jobs[0].JobID != job.JobID {
		t.Fatalf("expected agent job work notification, got %+v", publisher.jobs)
	}
}

type recordingWorkQueuePublisher struct {
	outbox []model.OutboxDelivery
	jobs   []model.AgentJob
	err    error
}

func (p *recordingWorkQueuePublisher) PublishOutboxDelivery(_ context.Context, delivery model.OutboxDelivery) error {
	p.outbox = append(p.outbox, delivery)
	return p.err
}

func (p *recordingWorkQueuePublisher) PublishAgentJob(_ context.Context, job model.AgentJob) error {
	p.jobs = append(p.jobs, job)
	return p.err
}
