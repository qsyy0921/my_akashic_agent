package outboxeventstore_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/outboxeventstore"
)

func TestOutboxDeliveryEventStoreAppendsAndListsEvents(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "outbox-events.jsonl")
	events, err := outboxeventstore.NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	for _, spec := range []struct {
		id        string
		delivery  string
		eventType model.OutboxDeliveryEventType
		status    model.DeliveryStatus
		at        time.Time
	}{
		{"event-1", "delivery-1", model.OutboxDeliveryEventQueued, model.DeliveryQueued, now},
		{"event-2", "delivery-1", model.OutboxDeliveryEventLeased, model.DeliveryDispatching, now.Add(time.Second)},
		{"event-3", "delivery-2", model.OutboxDeliveryEventQueued, model.DeliveryQueued, now.Add(2 * time.Second)},
	} {
		event, err := model.NewOutboxDeliveryEvent(model.OutboxDeliveryEventSpec{
			EventID:     spec.id,
			DeliveryID:  spec.delivery,
			Channel:     sampleOutboxEventChannel(),
			EventType:   spec.eventType,
			Status:      spec.status,
			Attempt:     1,
			MaxAttempts: 3,
		}, spec.at)
		if err != nil {
			t.Fatalf("new event: %v", err)
		}
		if err := events.AppendOutboxDeliveryEvent(ctx, event); err != nil {
			t.Fatalf("append event: %v", err)
		}
	}

	reloaded, err := outboxeventstore.NewStore(path)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	items, err := reloaded.ListOutboxDeliveryEvents(ctx, query.OutboxDeliveryEventFilter{
		DeliveryID: "delivery-1",
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 events, got %d", len(items))
	}
	if items[0].EventID != "event-2" || items[1].EventID != "event-1" {
		t.Fatalf("events should be newest-first, got %#v", items)
	}
}

func sampleOutboxEventChannel() model.ChannelRef {
	return model.ChannelRef{
		Kind:             model.ChannelKindTelegram,
		AccountID:        "bot",
		ConversationID:   "chat",
		ConversationType: model.ConversationTypePrivate,
	}
}
