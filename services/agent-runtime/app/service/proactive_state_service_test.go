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
}
