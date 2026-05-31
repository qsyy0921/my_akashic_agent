package inbounddedupestore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func TestStorePersistsInboundDedupeRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inbound-dedupe.json")
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	record, err := model.NewInboundDedupeRecord(model.InboundDedupeSpec{
		Scope:      "telegram:telegram",
		MessageKey: "123:456",
		SeenAt:     time.Date(2026, 5, 31, 11, 0, 0, 0, time.UTC),
		ExpiresAt:  time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveInboundDedupeRecord(context.Background(), record); err != nil {
		t.Fatal(err)
	}

	reopened, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	found, ok, err := reopened.FindInboundDedupeRecord(context.Background(), "telegram:telegram", "123:456")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || found.MessageKey != "123:456" {
		t.Fatalf("expected persisted record, got ok=%t record=%#v", ok, found)
	}
}

func TestStoreDeletesExpiredInboundDedupeRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inbound-dedupe.json")
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	record, err := model.NewInboundDedupeRecord(model.InboundDedupeSpec{
		Scope:      "telegram:telegram",
		MessageKey: "123:456",
		SeenAt:     time.Date(2026, 5, 31, 11, 0, 0, 0, time.UTC),
		ExpiresAt:  time.Date(2026, 5, 31, 11, 1, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveInboundDedupeRecord(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	deleted, err := store.DeleteExpiredInboundDedupeRecords(context.Background(), time.Date(2026, 5, 31, 11, 2, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}
	items, err := store.ListInboundDedupeRecords(context.Background(), query.InboundDedupeFilter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty records, got %#v", items)
	}
}
