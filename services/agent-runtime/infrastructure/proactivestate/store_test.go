package proactivestate_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/proactivestate"
)

func TestStorePersistsProactiveSchedulingState(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "proactive-state.json")
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)

	store, err := proactivestate.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	delivery, err := model.NewProactiveDeliveryRecord("telegram:1", "delivery-a", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProactiveDelivery(ctx, delivery); err != nil {
		t.Fatal(err)
	}
	seen, err := model.NewProactiveSeenItemRecord("mcp:news:feed-a", "item-a", now.Add(30*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProactiveSeenItems(ctx, []model.ProactiveSeenItemRecord{seen}); err != nil {
		t.Fatal(err)
	}
	rejection, err := model.NewProactiveRejectionCooldownRecord("qq:group:1", "item-b", now.Add(45*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProactiveRejectionCooldowns(ctx, []model.ProactiveRejectionCooldownRecord{rejection}); err != nil {
		t.Fatal(err)
	}
	contextOnly, err := model.NewProactiveContextOnlyRecord("telegram:1", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProactiveContextOnly(ctx, contextOnly); err != nil {
		t.Fatal(err)
	}
	mark, err := model.NewProactiveSessionMark("telegram:1", model.ProactiveSessionMarkDriftLastAt, now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProactiveSessionMark(ctx, mark); err != nil {
		t.Fatal(err)
	}
	globalMark, err := model.NewProactiveGlobalMark(model.ProactiveGlobalMarkBGContextMainAt, now.Add(150*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProactiveGlobalMark(ctx, globalMark); err != nil {
		t.Fatal(err)
	}
	quota, err := model.NewProactiveAnyActionQuota("default", "2026-05-30@12@Asia/Shanghai", now.Add(24*time.Hour), 2, now.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProactiveAnyActionQuota(ctx, quota); err != nil {
		t.Fatal(err)
	}

	reloaded, err := proactivestate.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	found, ok, err := reloaded.FindProactiveDelivery(ctx, "telegram:1", "delivery-a")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || found.DeliveryKey != "delivery-a" {
		t.Fatalf("expected persisted delivery, got ok=%v record=%+v", ok, found)
	}
	count, err := reloaded.CountProactiveDeliveriesSince(ctx, "telegram:1", now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("unexpected delivery count: %d", count)
	}
	foundSeen, ok, err := reloaded.FindProactiveSeenItem(ctx, "mcp:news:feed-b", "item-a")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || foundSeen.SourceKey != "mcp:news" {
		t.Fatalf("expected persisted normalized seen item, got ok=%v record=%+v", ok, foundSeen)
	}
	foundRejection, ok, err := reloaded.FindProactiveRejectionCooldown(ctx, "qq:group:1", "item-b")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || foundRejection.RejectedAt.IsZero() {
		t.Fatalf("expected persisted rejection cooldown, got ok=%v record=%+v", ok, foundRejection)
	}
	contextCount, err := reloaded.CountProactiveContextOnlySince(ctx, "telegram:1", now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if contextCount != 1 {
		t.Fatalf("unexpected context count: %d", contextCount)
	}
	lastDrift, ok, err := reloaded.FindProactiveSessionMark(ctx, "telegram:1", model.ProactiveSessionMarkDriftLastAt)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || lastDrift.MarkedAt.IsZero() {
		t.Fatalf("expected drift mark, got ok=%v mark=%+v", ok, lastDrift)
	}
	foundGlobal, ok, err := reloaded.FindProactiveGlobalMark(ctx, model.ProactiveGlobalMarkBGContextMainAt)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || foundGlobal.MarkedAt.IsZero() {
		t.Fatalf("expected global mark, got ok=%v mark=%+v", ok, foundGlobal)
	}
	listed, err := reloaded.ListProactiveDeliveries(ctx, query.ProactiveDeliveryFilter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("unexpected listed deliveries: %d", len(listed))
	}
	foundQuota, ok, err := reloaded.FindProactiveAnyActionQuota(ctx, "default")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || foundQuota.Used != 2 {
		t.Fatalf("expected persisted quota, got ok=%v quota=%+v", ok, foundQuota)
	}

	result, err := reloaded.CleanupProactiveState(ctx, model.ProactiveStateRetentionCutoffs{
		DeliveriesBefore:         now.Add(time.Hour),
		SeenItemsBefore:          now.Add(time.Hour),
		ContextOnlyBefore:        now.Add(time.Hour),
		RejectionCooldownsBefore: now.Add(time.Hour),
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
	cleaned, err := proactivestate.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, err := cleaned.FindProactiveSeenItem(ctx, "mcp:news", "item-a"); err != nil || ok {
		t.Fatalf("expected seen item cleanup, ok=%v err=%v", ok, err)
	}
}
