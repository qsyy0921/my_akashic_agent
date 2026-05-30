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
	listed, err := reloaded.ListProactiveDeliveries(ctx, query.ProactiveDeliveryFilter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("unexpected listed deliveries: %d", len(listed))
	}
}
