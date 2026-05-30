package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/memory"
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

func TestOutboxServiceLeaseNextMarksDeliveryDispatching(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 5, 30, 5, 0, 0, 0, time.UTC)
	delivery, err := model.NewOutboxDelivery(sampleOutboxMessage(now), 2, now)
	if err != nil {
		t.Fatalf("new delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, delivery); err != nil {
		t.Fatalf("save delivery: %v", err)
	}
	if err := store.EnqueueOutboxDelivery(ctx, delivery); err != nil {
		t.Fatalf("enqueue delivery: %v", err)
	}

	service := appservice.NewOutboxService(store, store)
	leased, err := service.LeaseNext(ctx, command.LeaseNextOutboxCommand{
		WorkerID:   "qq-dispatcher",
		TTLSeconds: 60,
		Timestamp:  now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("lease next: %v", err)
	}
	if leased.Status != string(model.DeliveryDispatching) {
		t.Fatalf("expected dispatching, got %s", leased.Status)
	}
	if leased.Attempts != 1 {
		t.Fatalf("expected attempts to increment, got %d", leased.Attempts)
	}
	if leased.LeaseOwner != "qq-dispatcher" {
		t.Fatalf("expected lease owner, got %q", leased.LeaseOwner)
	}
	if leased.LeaseExpiresAt == "" {
		t.Fatalf("expected lease expiry")
	}

	_, err = service.LeaseNext(ctx, command.LeaseNextOutboxCommand{
		WorkerID:   "qq-dispatcher-2",
		TTLSeconds: 60,
		Timestamp:  now.Add(2 * time.Second),
	})
	if err == nil {
		t.Fatalf("expected no leaseable delivery while lease is active")
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
