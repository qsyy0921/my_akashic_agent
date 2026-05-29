package sendledgerstore_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/service"
	store "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/sendledgerstore"
)

func TestSendLedgerStorePersistsRecordsAcrossRestarts(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "send-ledger.json")
	repo, err := store.NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	first := newRecord("1049511700", "2365524513", "hello", time.Now().UTC())
	second := newRecord("2365524513", "1049511700", "world", time.Now().UTC().Add(time.Second))
	if err := repo.RecordSent(ctx, first); err != nil {
		t.Fatalf("record first: %v", err)
	}
	if err := repo.RecordSent(ctx, second); err != nil {
		t.Fatalf("record second: %v", err)
	}

	reloaded, err := store.NewStore(path)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	if !reloaded.RecentlySent("1049511700", "2365524513", first.ContentHash, time.Minute) {
		t.Fatal("expected first record to be recent after reload")
	}
	items, err := reloaded.ListSentRecords(ctx, query.SendRecordFilter{Limit: 10})
	if err != nil {
		t.Fatalf("list records: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 records, got %d", len(items))
	}
	if items[0].ContentHash != second.ContentHash {
		t.Fatalf("expected newest record first, got %+v", items)
	}
	filtered, err := reloaded.ListSentRecords(ctx, query.SendRecordFilter{
		Limit:          10,
		FromBotID:      "1049511700",
		ConversationID: "2365524513",
	})
	if err != nil {
		t.Fatalf("list filtered: %v", err)
	}
	if len(filtered) != 1 || filtered[0].ContentHash != first.ContentHash {
		t.Fatalf("unexpected filtered records: %+v", filtered)
	}
}

func TestSendLedgerStoreRejectsInvalidRecord(t *testing.T) {
	repo, err := store.NewStore(filepath.Join(t.TempDir(), "send-ledger.json"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	err = repo.RecordSent(context.Background(), model.SendRecord{})
	if err == nil {
		t.Fatal("expected invalid record error")
	}
}

func newRecord(fromBotID string, conversationID string, content string, timestamp time.Time) model.SendRecord {
	return model.SendRecord{
		FromBotID:      fromBotID,
		ConversationID: conversationID,
		ContentHash:    service.ContentHash(content),
		Timestamp:      timestamp,
	}
}
