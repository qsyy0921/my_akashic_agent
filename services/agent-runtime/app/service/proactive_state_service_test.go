package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/memory"
)

func TestProactiveStateServiceRecordsDeliveryAndChecksWindow(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	svc := service.NewProactiveStateService(memory.NewStore())

	if _, err := svc.RecordDelivery(ctx, command.RecordProactiveDeliveryCommand{
		SessionKey:  "telegram:1",
		DeliveryKey: "delivery-a",
		Timestamp:   now,
	}); err != nil {
		t.Fatal(err)
	}
	duplicate, err := svc.IsDeliveryDuplicate(ctx, command.CheckProactiveDeliveryDuplicateCommand{
		SessionKey:  "telegram:1",
		DeliveryKey: "delivery-a",
		WindowHours: 1,
		Timestamp:   now.Add(30 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !duplicate.Duplicate {
		t.Fatal("expected duplicate in one hour window")
	}
	count, err := svc.CountDeliveries(ctx, command.CountProactiveDeliveriesCommand{
		SessionKey:  "telegram:1",
		WindowHours: 24,
		Timestamp:   now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if count.Count != 1 {
		t.Fatalf("unexpected delivery count: %d", count.Count)
	}
}

func TestProactiveStateServiceRecordsSeenAndRejectionCooldown(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	svc := service.NewProactiveStateService(memory.NewStore())

	marked, err := svc.MarkItemsSeen(ctx, command.MarkProactiveItemsSeenCommand{
		Entries: []command.ProactiveSourceItemEntry{{
			SourceKey: "mcp:news:feed-a",
			ItemID:    "item-a",
		}},
		Timestamp: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if marked.Count != 1 {
		t.Fatalf("unexpected seen count: %d", marked.Count)
	}
	seen, err := svc.IsItemSeen(ctx, command.CheckProactiveItemSeenCommand{
		SourceKey: "mcp:news:feed-b",
		ItemID:    "item-a",
		TTLHours:  2,
		Timestamp: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !seen.Seen || seen.SourceKey != "mcp:news" {
		t.Fatalf("expected normalized seen hit, got %+v", seen)
	}
	expired, err := svc.IsItemSeen(ctx, command.CheckProactiveItemSeenCommand{
		SourceKey: "mcp:news:feed-b",
		ItemID:    "item-a",
		TTLHours:  1,
		Timestamp: now.Add(2 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if expired.Seen {
		t.Fatalf("expected seen item to expire, got %+v", expired)
	}

	if _, err := svc.MarkRejectionCooldown(ctx, command.MarkProactiveRejectionCooldownCommand{
		Entries: []command.ProactiveSourceItemEntry{{
			SourceKey: "qq:group:1",
			ItemID:    "item-b",
		}},
		Hours:     3,
		Timestamp: now,
	}); err != nil {
		t.Fatal(err)
	}
	cooled, err := svc.IsRejectionCooled(ctx, command.CheckProactiveRejectionCooldownCommand{
		SourceKey: "qq:group:1",
		ItemID:    "item-b",
		TTLHours:  3,
		Timestamp: now.Add(2 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !cooled.Cooled {
		t.Fatalf("expected rejection cooldown hit, got %+v", cooled)
	}
}

func TestProactiveStateServiceCleanupPrunesExpiredRuntimeState(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	svc := service.NewProactiveStateService(memory.NewStore())

	if _, err := svc.RecordDelivery(ctx, command.RecordProactiveDeliveryCommand{
		SessionKey:  "telegram:1",
		DeliveryKey: "old-delivery",
		Timestamp:   now.Add(-3 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RecordDelivery(ctx, command.RecordProactiveDeliveryCommand{
		SessionKey:  "telegram:1",
		DeliveryKey: "fresh-delivery",
		Timestamp:   now.Add(-30 * time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MarkItemsSeen(ctx, command.MarkProactiveItemsSeenCommand{
		Entries:   []command.ProactiveSourceItemEntry{{SourceKey: "mcp:news:feed", ItemID: "old-item"}},
		Timestamp: now.Add(-3 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RecordContextOnly(ctx, command.RecordProactiveContextOnlyCommand{
		SessionKey: "telegram:1",
		Timestamp:  now.Add(-3 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MarkRejectionCooldown(ctx, command.MarkProactiveRejectionCooldownCommand{
		Entries:   []command.ProactiveSourceItemEntry{{SourceKey: "qq:group:1", ItemID: "old-rejection"}},
		Hours:     2,
		Timestamp: now.Add(-3 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	result, err := svc.Cleanup(ctx, command.CleanupProactiveStateCommand{
		SeenTTLHours:              1,
		DeliveryTTLHours:          1,
		ContextOnlyTTLHours:       1,
		RejectionCooldownTTLHours: 1,
		Timestamp:                 now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.RemovedDeliveries != 1 ||
		result.RemovedSeenItems != 1 ||
		result.RemovedContextOnly != 1 ||
		result.RemovedRejectionCooldowns != 1 {
		t.Fatalf("unexpected cleanup result: %+v", result)
	}
	count, err := svc.CountDeliveries(ctx, command.CountProactiveDeliveriesCommand{
		SessionKey:  "telegram:1",
		WindowHours: 24,
		Timestamp:   now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if count.Count != 1 {
		t.Fatalf("expected fresh delivery to remain, got %d", count.Count)
	}
}

func TestProactiveStateServiceRecordsContextAndDriftMarks(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	svc := service.NewProactiveStateService(memory.NewStore())

	if _, err := svc.RecordContextOnly(ctx, command.RecordProactiveContextOnlyCommand{
		SessionKey: "telegram:1",
		Timestamp:  now,
	}); err != nil {
		t.Fatal(err)
	}
	lastContext, err := svc.LastContextOnly(ctx, "telegram:1")
	if err != nil {
		t.Fatal(err)
	}
	if !lastContext.Found || lastContext.Timestamp == "" {
		t.Fatalf("expected context mark, got %+v", lastContext)
	}
	contextCount, err := svc.CountContextOnly(ctx, command.CountProactiveContextOnlyCommand{
		SessionKey:  "telegram:1",
		WindowHours: 24,
		Timestamp:   now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if contextCount.Count != 1 {
		t.Fatalf("unexpected context count: %d", contextCount.Count)
	}

	if _, err := svc.RecordDriftRun(ctx, command.RecordProactiveDriftRunCommand{
		SessionKey: "telegram:1",
		Timestamp:  now.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	lastDrift, err := svc.LastDriftRun(ctx, "telegram:1")
	if err != nil {
		t.Fatal(err)
	}
	if !lastDrift.Found || lastDrift.Timestamp == "" {
		t.Fatalf("expected drift mark, got %+v", lastDrift)
	}

	if _, err := svc.RecordBGContextMain(ctx, command.RecordProactiveBGContextMainCommand{
		Timestamp: now.Add(2 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	lastBGContext, err := svc.LastBGContextMain(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !lastBGContext.Found ||
		lastBGContext.Key != "bg_context_last_main_at" ||
		lastBGContext.Timestamp == "" {
		t.Fatalf("expected bg context mark, got %+v", lastBGContext)
	}
}

func TestProactiveStateServiceAnyActionQuotaRolloverAndRecord(t *testing.T) {
	ctx := context.Background()
	svc := service.NewProactiveStateService(memory.NewStore())
	now := time.Date(2026, 5, 30, 3, 0, 0, 0, time.UTC)

	snapshot, err := svc.SnapshotAnyActionQuota(ctx, command.SnapshotProactiveAnyActionQuotaCommand{
		ResetHour: 12,
		Timezone:  "Asia/Shanghai",
		Timestamp: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.WindowKey != "2026-05-29@12@Asia/Shanghai" {
		t.Fatalf("unexpected window key: %s", snapshot.WindowKey)
	}
	if snapshot.Used != 0 {
		t.Fatalf("unexpected initial used: %d", snapshot.Used)
	}

	recorded, err := svc.RecordAnyAction(ctx, command.RecordProactiveAnyActionCommand{
		ResetHour: 12,
		Timezone:  "Asia/Shanghai",
		Timestamp: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if recorded.Used != 1 || recorded.LastActionAt == "" {
		t.Fatalf("expected recorded quota, got %+v", recorded)
	}

	rolled, err := svc.SnapshotAnyActionQuota(ctx, command.SnapshotProactiveAnyActionQuotaCommand{
		ResetHour: 12,
		Timezone:  "Asia/Shanghai",
		Timestamp: now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if rolled.Used != 0 {
		t.Fatalf("expected rollover to reset used, got %d", rolled.Used)
	}
	if rolled.LastActionAt == "" {
		t.Fatalf("expected rollover to preserve last action timestamp")
	}
}
