package receiverstatusstore_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/receiverstatusstore"
)

func TestReceiverStatusStorePersistsLatestStatus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receiver-statuses.json")
	store, err := receiverstatusstore.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveReceiverStatus(context.Background(), newReceiverStatus(t, "qq", "qq", "1049511700", "connected")); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveReceiverStatus(context.Background(), newReceiverStatus(t, "qq", "qq", "1049511700", "stopped")); err != nil {
		t.Fatal(err)
	}

	reopened, err := receiverstatusstore.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	items, err := reopened.ListReceiverStatuses(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one receiver, got %d", len(items))
	}
	if items[0].Status != model.ReceiverStatusStopped {
		t.Fatalf("expected latest status, got %#v", items[0])
	}
}

func TestReceiverStatusStoreSortsReceivers(t *testing.T) {
	store, err := receiverstatusstore.NewStore(filepath.Join(t.TempDir(), "receiver-statuses.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []model.ReceiverStatus{
		newReceiverStatus(t, "telegram", "telegram", "telegram", "connected"),
		newReceiverStatus(t, "qq", "qq_2365524513", "2365524513", "connected"),
		newReceiverStatus(t, "qq", "qq", "1049511700", "connected"),
	} {
		if err := store.SaveReceiverStatus(context.Background(), item); err != nil {
			t.Fatal(err)
		}
	}

	items, err := store.ListReceiverStatuses(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got := []string{items[0].ReceiverID, items[1].ReceiverID, items[2].ReceiverID}
	want := []string{"qq:1049511700:qq", "qq:2365524513:qq_2365524513", "telegram:telegram:telegram"}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("unexpected order: got %#v want %#v", got, want)
		}
	}
}

func TestReceiverStatusStoreDeletesReceiver(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receiver-statuses.json")
	store, err := receiverstatusstore.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	first := newReceiverStatus(t, "qq", "qq", "1049511700", "connected")
	second := newReceiverStatus(t, "telegram", "telegram", "telegram", "connected")
	if err := store.SaveReceiverStatus(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveReceiverStatus(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteReceiverStatus(context.Background(), first.ReceiverID); err != nil {
		t.Fatal(err)
	}

	reopened, err := receiverstatusstore.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	items, err := reopened.ListReceiverStatuses(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ReceiverID != second.ReceiverID {
		t.Fatalf("expected only second receiver to remain, got %#v", items)
	}
}

func newReceiverStatus(t *testing.T, kind string, channelName string, accountID string, status string) model.ReceiverStatus {
	t.Helper()
	item, err := model.NewReceiverStatus(model.ReceiverStatusSpec{
		Kind:        model.ChannelKind(kind),
		ChannelName: channelName,
		AccountID:   accountID,
		Status:      status,
		Reason:      "test",
		Source:      "test",
	}, time.Date(2026, 5, 31, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return item
}
