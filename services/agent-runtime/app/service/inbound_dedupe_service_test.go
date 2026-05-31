package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
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
