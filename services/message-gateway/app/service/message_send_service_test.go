package service_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/app/command"
	appservice "github.com/kachofugetsu09/akashic-agent/services/message-gateway/app/service"
	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/domain/service"
	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/infrastructure/memory"
)

func TestMessageSendServiceAddsBotProtocolAndRecordsLedger(t *testing.T) {
	store := memory.NewStore()
	sender := appservice.NewMessageSendService(store, store)

	err := sender.Send(context.Background(), command.SendMessageCommand{
		EventID: "outbound-1",
		Channel: command.ChannelCommand{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   "2365524513",
			ConversationType: "private",
		},
		Content:         "reply from bot",
		Timestamp:       time.Now().UTC(),
		WithBotProtocol: true,
		ProtocolNonce:   "nonce-1",
		ProtocolNextHop: 2,
	})
	if err != nil {
		t.Fatalf("send returned error: %v", err)
	}

	outbound := store.Outbound()
	if len(outbound) != 1 {
		t.Fatalf("expected 1 outbound message, got %d", len(outbound))
	}
	if !strings.HasPrefix(outbound[0].Content, "[[akashic:bot") {
		t.Fatalf("expected bot protocol tag, got %q", outbound[0].Content)
	}
	if !store.RecentlySent("1049511700", "2365524513", service.ContentHash(outbound[0].Content), time.Minute) {
		t.Fatal("expected send ledger to contain outbound content hash")
	}
}
