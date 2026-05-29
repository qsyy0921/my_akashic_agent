package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/app/command"
	appservice "github.com/kachofugetsu09/akashic-agent/services/message-gateway/app/service"
	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/infrastructure/memory"
)

func TestOutboxServiceTransitionsAndRetry(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 5, 30, 4, 0, 0, 0, time.UTC)
	delivery, err := model.NewOutboxDelivery(sampleOutboxMessage(now), 2, now)
	if err != nil {
		t.Fatalf("new delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, delivery); err != nil {
		t.Fatalf("save delivery: %v", err)
	}

	service := appservice.NewOutboxService(store, store)
	running, err := service.MarkDispatching(ctx, command.MarkOutboxDispatchingCommand{
		EventID:   "outbox-1",
		Timestamp: now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("mark dispatching: %v", err)
	}
	if running.Status != string(model.DeliveryDispatching) || running.Attempts != 1 {
		t.Fatalf("unexpected running state: %+v", running)
	}

	failed, err := service.MarkFailed(ctx, command.MarkOutboxFailedCommand{
		EventID:      "outbox-1",
		ErrorMessage: "platform timeout",
		Timestamp:    now.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	if failed.Status != string(model.DeliveryFailed) {
		t.Fatalf("expected failed, got %s", failed.Status)
	}

	retried, err := service.Retry(ctx, command.RetryOutboxCommand{
		EventID:   "outbox-1",
		Timestamp: now.Add(3 * time.Second),
	})
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if retried.Status != string(model.DeliveryQueued) {
		t.Fatalf("expected queued, got %s", retried.Status)
	}
	if len(store.OutboxQueue()) != 1 {
		t.Fatalf("expected retried delivery to be enqueued, got %d", len(store.OutboxQueue()))
	}
}

func sampleOutboxMessage(timestamp time.Time) model.OutboundMessage {
	return model.OutboundMessage{
		EventID: "outbox-1",
		Channel: model.ChannelRef{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   "2365524513",
			ConversationType: "private",
		},
		Content:   "hello",
		Timestamp: timestamp,
	}
}
