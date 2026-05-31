package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/inbounddedupestore"
)

func TestInboundDedupeServiceDetectsDuplicates(t *testing.T) {
	store, err := inbounddedupestore.NewStore(filepath.Join(t.TempDir(), "inbound-dedupe.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := NewInboundDedupeService(store)
	now := time.Date(2026, 5, 31, 11, 0, 0, 0, time.UTC)
	first, err := service.Check(context.Background(), command.CheckInboundDedupeCommand{
		Scope:      "telegram:telegram",
		MessageKey: "123:456",
		TTLSeconds: 60,
		Timestamp:  now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Duplicate || first.SeenCount != 1 {
		t.Fatalf("unexpected first result: %#v", first)
	}
	second, err := service.Check(context.Background(), command.CheckInboundDedupeCommand{
		Scope:      "telegram:telegram",
		MessageKey: "123:456",
		TTLSeconds: 60,
		Timestamp:  now.Add(time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !second.Duplicate || second.SeenCount != 2 {
		t.Fatalf("expected duplicate result: %#v", second)
	}
}

func TestInboundDedupeServiceTreatsExpiredRecordAsNew(t *testing.T) {
	store, err := inbounddedupestore.NewStore(filepath.Join(t.TempDir(), "inbound-dedupe.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := NewInboundDedupeService(store)
	now := time.Date(2026, 5, 31, 11, 0, 0, 0, time.UTC)
	_, err = service.Check(context.Background(), command.CheckInboundDedupeCommand{
		Scope:      "telegram:telegram",
		MessageKey: "123:456",
		TTLSeconds: 30,
		Timestamp:  now,
	})
	if err != nil {
		t.Fatal(err)
	}
	later, err := service.Check(context.Background(), command.CheckInboundDedupeCommand{
		Scope:      "telegram:telegram",
		MessageKey: "123:456",
		TTLSeconds: 30,
		Timestamp:  now.Add(31 * time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}
	if later.Duplicate || later.SeenCount != 1 {
		t.Fatalf("expected expired record to be new: %#v", later)
	}
}

func TestInboundDedupeServiceMetricsSummarizesDuplicateSuppression(t *testing.T) {
	store, err := inbounddedupestore.NewStore(filepath.Join(t.TempDir(), "inbound-dedupe.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := NewInboundDedupeService(store)
	now := time.Date(2100, 5, 31, 11, 0, 0, 0, time.UTC)
	for i := 0; i < 2; i++ {
		if _, err := service.Check(context.Background(), command.CheckInboundDedupeCommand{
			Scope:      "qq:qq_2365524513:2365524513",
			MessageKey: "group:27234224:1194",
			TTLSeconds: 60,
			Timestamp:  now.Add(time.Duration(i) * time.Second),
		}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := service.Check(context.Background(), command.CheckInboundDedupeCommand{
		Scope:      "telegram:telegram",
		MessageKey: "123:456",
		TTLSeconds: 60,
		Timestamp:  now,
	}); err != nil {
		t.Fatal(err)
	}

	metrics, err := service.Metrics(context.Background(), query.InboundDedupeMetricsFilter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if metrics.SampledRecords != 2 || metrics.DuplicateRecords != 1 || metrics.DuplicateSeenTotal != 1 || metrics.SeenTotal != 3 {
		t.Fatalf("unexpected metrics: %#v", metrics)
	}
	if len(metrics.Scopes) != 2 || metrics.Scopes[0].Scope != "qq:qq_2365524513:2365524513" {
		t.Fatalf("unexpected scope metrics: %#v", metrics.Scopes)
	}
	if metrics.SideEffect != "none" || metrics.Totals["scopes"] != 2 {
		t.Fatalf("unexpected metrics metadata: %#v", metrics)
	}
}
