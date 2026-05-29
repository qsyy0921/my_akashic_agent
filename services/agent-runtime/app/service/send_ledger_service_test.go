package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/service"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/memory"
)

func TestSendLedgerServiceRecordsAndQueriesRecentSend(t *testing.T) {
	store := memory.NewStore()
	ledger := appservice.NewSendLedgerService(store)
	ctx := context.Background()
	now := time.Now().UTC()

	record, err := ledger.Record(ctx, command.RecordSendCommand{
		FromBotID:      "1049511700",
		ConversationID: "2365524513",
		Content:        "generate one image",
		Timestamp:      now,
	})
	if err != nil {
		t.Fatalf("record send: %v", err)
	}
	expectedHash := service.ContentHash("generate one image")
	if record.ContentHash != expectedHash {
		t.Fatalf("expected hash %s, got %s", expectedHash, record.ContentHash)
	}

	recent, err := ledger.RecentlySent(ctx, command.CheckRecentSendCommand{
		FromBotID:      "1049511700",
		ConversationID: "2365524513",
		Content:        "generate one image",
		Window:         time.Minute,
	})
	if err != nil {
		t.Fatalf("recently sent: %v", err)
	}
	if !recent.Recent {
		t.Fatal("expected recent send to be detected")
	}

	items, err := ledger.List(ctx, query.SendRecordFilter{Limit: 10})
	if err != nil {
		t.Fatalf("list records: %v", err)
	}
	if len(items) != 1 || items[0].ContentHash != expectedHash {
		t.Fatalf("unexpected records: %+v", items)
	}
}
