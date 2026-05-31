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
	driftState, err := model.NewProactiveDriftSkillState("explore-curiosity", now.Add(4*time.Minute), 3, model.ProactiveDriftSkillStatusInProgress, "next")
	if err != nil {
		t.Fatal(err)
	}
	driftRun, err := model.NewProactiveDriftRecentRun("explore-curiosity", now.Add(4*time.Minute), "did work", model.ProactiveDriftMessageResultSilent)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProactiveDriftFinish(ctx, driftState, driftRun, "note", 10); err != nil {
		t.Fatal(err)
	}
	tick, err := model.NewProactiveTickLogFinish(
		"tick-1",
		"telegram:1",
		now.Add(5*time.Minute),
		now.Add(5*time.Minute+time.Second),
		"",
		"reply",
		"",
		1,
		0,
		1,
		0,
		[]string{"feed:1"},
		nil,
		[]string{"feed:1"},
		false,
		"hello",
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProactiveTickLogFinish(ctx, tick, 500); err != nil {
		t.Fatal(err)
	}
	step, err := model.NewProactiveTickStepLog(
		"tick-1",
		1,
		"loop",
		"message_push",
		"call-1",
		map[string]any{"message": "hello"},
		`{"ok":true}`,
		"reply",
		"",
		[]string{"feed:1"},
		nil,
		[]string{"feed:1"},
		"hello",
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProactiveTickStepLog(ctx, step, 5000); err != nil {
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
	foundDrift, ok, err := reloaded.FindProactiveDriftSkillState(ctx, "explore-curiosity")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || foundDrift.RunCount != 3 || foundDrift.Next != "next" {
		t.Fatalf("expected persisted drift skill state, got ok=%v state=%+v", ok, foundDrift)
	}
	driftRuns, driftNote, err := reloaded.ListProactiveDriftRecentRuns(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if driftNote != "note" || len(driftRuns) != 1 || driftRuns[0].OneLine != "did work" {
		t.Fatalf("expected persisted drift runs, note=%q runs=%+v", driftNote, driftRuns)
	}
	tickLogs, totalTicks, err := reloaded.ListProactiveTickLogs(ctx, query.ProactiveTickLogFilter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if totalTicks != 1 || len(tickLogs) != 1 || tickLogs[0].FinalMessage != "hello" {
		t.Fatalf("expected persisted tick log, total=%d logs=%+v", totalTicks, tickLogs)
	}
	tickSteps, err := reloaded.ListProactiveTickStepLogs(ctx, "tick-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(tickSteps) != 1 || tickSteps[0].ToolName != "message_push" {
		t.Fatalf("expected persisted tick step, steps=%+v", tickSteps)
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

func TestStoreListsProactiveTickLogsWithDashboardFilters(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "proactive-state.json")
	store, err := proactivestate.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 5, 31, 10, 0, 0, 0, time.UTC)
	for idx, tickID := range []string{"tick-1", "tick-2", "tick-3"} {
		log, err := model.NewProactiveTickLogFinish(
			tickID,
			"telegram:1",
			base.Add(time.Duration(idx)*time.Minute),
			base.Add(time.Duration(idx)*time.Minute+time.Second),
			"",
			"reply",
			"",
			idx,
			0,
			0,
			0,
			nil,
			nil,
			nil,
			tickID == "tick-3",
			"",
		)
		if err != nil {
			t.Fatal(err)
		}
		if err := store.SaveProactiveTickLogFinish(ctx, log, 10); err != nil {
			t.Fatal(err)
		}
	}

	items, total, err := store.ListProactiveTickLogs(ctx, query.ProactiveTickLogFilter{
		Limit:       1,
		Offset:      1,
		StartedFrom: base.Add(time.Minute),
		StartedTo:   base.Add(2 * time.Minute),
		SortBy:      "started_at",
		SortOrder:   "asc",
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(items) != 1 || items[0].TickID != "tick-3" {
		t.Fatalf("unexpected paged tick logs: total=%d items=%+v", total, items)
	}

	drift, driftTotal, err := store.ListProactiveTickLogs(ctx, query.ProactiveTickLogFilter{
		Limit: 10,
		Flow:  "drift",
	})
	if err != nil {
		t.Fatal(err)
	}
	if driftTotal != 1 || len(drift) != 1 || drift[0].TickID != "tick-3" {
		t.Fatalf("unexpected drift tick logs: total=%d items=%+v", driftTotal, drift)
	}
}
