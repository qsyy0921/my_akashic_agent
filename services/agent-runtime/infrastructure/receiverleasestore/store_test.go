package receiverleasestore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func TestStorePersistsReceiverLeases(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receiver-leases.json")
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	lease, err := model.NewReceiverLease(model.ReceiverLeaseSpec{
		Kind:        model.ChannelKindTelegram,
		ChannelName: "telegram",
		AccountID:   "7689386159",
		HolderID:    "python:1",
		LeaseToken:  "tok-1",
		ExpiresAt:   time.Date(2026, 5, 31, 1, 5, 0, 0, time.UTC),
		AcquiredAt:  time.Date(2026, 5, 31, 1, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 5, 31, 1, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveReceiverLease(context.Background(), lease); err != nil {
		t.Fatal(err)
	}

	reopened, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	items, err := reopened.ListReceiverLeases(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ReceiverID != lease.ReceiverID || items[0].LeaseToken != "tok-1" {
		t.Fatalf("unexpected leases: %#v", items)
	}
}

func TestStoreDeletesReceiverLease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receiver-leases.json")
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	lease, err := model.NewReceiverLease(model.ReceiverLeaseSpec{
		Kind:        model.ChannelKindTelegram,
		ChannelName: "telegram",
		AccountID:   "7689386159",
		HolderID:    "python:1",
		LeaseToken:  "tok-1",
		ExpiresAt:   time.Now().UTC().Add(time.Minute),
		UpdatedAt:   time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveReceiverLease(context.Background(), lease); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteReceiverLease(context.Background(), lease.ReceiverID); err != nil {
		t.Fatal(err)
	}
	items, err := store.ListReceiverLeases(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected deleted lease, got %#v", items)
	}
}
