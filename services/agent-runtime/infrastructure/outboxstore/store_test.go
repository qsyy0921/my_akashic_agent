package outboxstore_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	store "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/outboxstore"
)

func TestOutboxStorePersistsDeliveriesAcrossRestarts(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "outbox.json")
	repo, err := store.NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	now := time.Date(2026, 5, 30, 9, 0, 0, 0, time.UTC)
	deliveryA := newDelivery(t, "outbox:a", "2365524513", now)
	deliveryB := newDelivery(t, "outbox:b", "1049511700", now.Add(time.Minute))
	if err := repo.SaveOutboxDelivery(ctx, deliveryA); err != nil {
		t.Fatalf("save deliveryA: %v", err)
	}
	if err := deliveryB.MarkDispatching(now.Add(2 * time.Minute)); err != nil {
		t.Fatalf("mark dispatching: %v", err)
	}
	if err := deliveryB.MarkFailedWithKind(model.DeliveryErrorPlatformTimeout, "platform timeout", now.Add(3*time.Minute)); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	if err := repo.SaveOutboxDelivery(ctx, deliveryB); err != nil {
		t.Fatalf("save deliveryB: %v", err)
	}
	if err := repo.EnqueueOutboxDelivery(ctx, deliveryB); err != nil {
		t.Fatalf("enqueue deliveryB: %v", err)
	}

	reloaded, err := store.NewStore(path)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	stored, ok, err := reloaded.FindOutboxDelivery(ctx, "outbox:b")
	if err != nil {
		t.Fatalf("find stored delivery: %v", err)
	}
	if !ok {
		t.Fatalf("expected outbox:b to persist")
	}
	if stored.Status != model.DeliveryFailed {
		t.Fatalf("expected failed status, got %s", stored.Status)
	}
	if stored.ErrorMessage != "platform timeout" {
		t.Fatalf("expected error message to persist, got %q", stored.ErrorMessage)
	}
	if stored.ErrorKind != model.DeliveryErrorPlatformTimeout {
		t.Fatalf("expected error kind to persist, got %q", stored.ErrorKind)
	}
	items, err := reloaded.ListOutboxDeliveries(ctx, 10)
	if err != nil {
		t.Fatalf("list deliveries: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 deliveries, got %d", len(items))
	}
	if items[0].Message.EventID != "outbox:b" {
		t.Fatalf("expected newest insertion first, got %+v", items)
	}
	queue := reloaded.QueueIDs()
	if len(queue) != 1 || queue[0] != "outbox:b" {
		t.Fatalf("expected queue to persist outbox:b, got %+v", queue)
	}
}

func TestOutboxStorePersistsUpdatedDeliveryState(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "outbox-update.json")
	repo, err := store.NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	now := time.Date(2026, 5, 30, 9, 30, 0, 0, time.UTC)
	delivery := newDelivery(t, "outbox:update", "2365524513", now)
	if err := repo.SaveOutboxDelivery(ctx, delivery); err != nil {
		t.Fatalf("save queued delivery: %v", err)
	}
	if err := delivery.MarkDispatching(now.Add(time.Minute)); err != nil {
		t.Fatalf("mark dispatching: %v", err)
	}
	if err := delivery.MarkSucceeded(now.Add(2 * time.Minute)); err != nil {
		t.Fatalf("mark succeeded: %v", err)
	}
	if err := repo.SaveOutboxDelivery(ctx, delivery); err != nil {
		t.Fatalf("save updated delivery: %v", err)
	}

	reloaded, err := store.NewStore(path)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	stored, ok, err := reloaded.FindOutboxDelivery(ctx, "outbox:update")
	if err != nil {
		t.Fatalf("find stored delivery: %v", err)
	}
	if !ok {
		t.Fatalf("expected stored delivery")
	}
	if stored.Status != model.DeliverySucceeded {
		t.Fatalf("expected succeeded status, got %s", stored.Status)
	}
	if stored.Attempts != 1 {
		t.Fatalf("expected attempts to persist, got %d", stored.Attempts)
	}
	items, err := reloaded.ListOutboxDeliveries(ctx, 10)
	if err != nil {
		t.Fatalf("list deliveries: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected idempotent delivery update, got %d items", len(items))
	}
}

func TestOutboxStoreFindLeaseableAndPersistsLease(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "outbox-lease.json")
	repo, err := store.NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	now := time.Date(2026, 5, 30, 11, 0, 0, 0, time.UTC)
	delivery := newDelivery(t, "outbox:lease", "2365524513", now)
	if err := repo.EnqueueOutboxDelivery(ctx, delivery); err != nil {
		t.Fatalf("enqueue delivery: %v", err)
	}
	leaseable, ok, err := repo.FindLeaseableOutboxDelivery(ctx, outport.OutboxLeaseFilter{Now: now.Add(time.Second)})
	if err != nil {
		t.Fatalf("find leaseable: %v", err)
	}
	if !ok {
		t.Fatalf("expected leaseable delivery")
	}
	if leaseable.Message.EventID != "outbox:lease" {
		t.Fatalf("unexpected leaseable delivery: %s", leaseable.Message.EventID)
	}
	if err := leaseable.Lease("qq-dispatcher", time.Minute, now.Add(2*time.Second)); err != nil {
		t.Fatalf("lease delivery: %v", err)
	}
	if err := repo.SaveOutboxDelivery(ctx, leaseable); err != nil {
		t.Fatalf("save leased delivery: %v", err)
	}

	reloaded, err := store.NewStore(path)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	stored, ok, err := reloaded.FindOutboxDelivery(ctx, "outbox:lease")
	if err != nil {
		t.Fatalf("find stored delivery: %v", err)
	}
	if !ok {
		t.Fatalf("expected leased delivery to persist")
	}
	if stored.LeaseOwner != "qq-dispatcher" {
		t.Fatalf("expected lease owner to persist, got %q", stored.LeaseOwner)
	}
	if stored.LeaseExpiresAt.IsZero() {
		t.Fatalf("expected lease expiry to persist")
	}
	if _, ok, err := reloaded.FindLeaseableOutboxDelivery(ctx, outport.OutboxLeaseFilter{Now: now.Add(30 * time.Second)}); err != nil {
		t.Fatalf("find active lease: %v", err)
	} else if ok {
		t.Fatalf("active lease should not be leaseable")
	}
	if _, ok, err := reloaded.FindLeaseableOutboxDelivery(ctx, outport.OutboxLeaseFilter{Now: now.Add(2 * time.Minute)}); err != nil {
		t.Fatalf("find expired lease: %v", err)
	} else if !ok {
		t.Fatalf("expired lease should be leaseable")
	}
}

func newDelivery(t *testing.T, eventID string, conversationID string, now time.Time) model.OutboxDelivery {
	t.Helper()
	delivery, err := model.NewOutboxDelivery(model.OutboundMessage{
		EventID: eventID,
		Channel: model.ChannelRef{
			Kind:             model.ChannelKindQQ,
			AccountID:        "1049511700",
			ConversationID:   conversationID,
			ConversationType: model.ConversationTypePrivate,
		},
		Content:   "generated image is ready",
		Timestamp: now,
		Metadata:  map[string]string{"source": "test"},
	}, 2, now)
	if err != nil {
		t.Fatalf("new outbox delivery: %v", err)
	}
	return delivery
}
