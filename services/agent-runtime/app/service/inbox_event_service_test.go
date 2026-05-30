package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/memory"
)

func TestInboxEventServiceListsRawGroupMessages(t *testing.T) {
	store := memory.NewStore()
	firstEvent, err := model.NewInboxEvent(model.MessageEnvelope{
		EventID: "qq:1049511700:group:27234224:1",
		Channel: model.ChannelRef{
			Kind:             model.ChannelKindQQ,
			AccountID:        "1049511700",
			ConversationID:   "27234224",
			ConversationType: model.ConversationTypeGroup,
		},
		Sender: model.SenderRef{
			ID:   "2948770636",
			Kind: model.SenderKindHuman,
		},
		Content:   "older group note",
		Timestamp: time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC),
		Metadata:  map[string]string{"observe_only": "true", "seq": "0"},
	}, model.LoopDecision{Action: model.LoopActionAllow, Reason: "accepted"}, time.Date(2026, 5, 30, 0, 0, 1, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	secondEvent, err := model.NewInboxEvent(model.MessageEnvelope{
		EventID: "qq:1049511700:group:27234224:2",
		Channel: model.ChannelRef{
			Kind:             model.ChannelKindQQ,
			AccountID:        "1049511700",
			ConversationID:   "27234224",
			ConversationType: model.ConversationTypeGroup,
		},
		Sender: model.SenderRef{
			ID:   "2948770636",
			Kind: model.SenderKindHuman,
		},
		Content:   "B850M board price screenshot",
		Timestamp: time.Date(2026, 5, 30, 0, 0, 2, 0, time.UTC),
		Metadata:  map[string]string{"observe_only": "true", "seq": "1"},
	}, model.LoopDecision{Action: model.LoopActionAllow, Reason: "accepted"}, time.Date(2026, 5, 30, 0, 0, 3, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveInboxEvent(context.Background(), firstEvent); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveInboxEvent(context.Background(), secondEvent); err != nil {
		t.Fatal(err)
	}

	service := appservice.NewInboxEventService(store)
	views, err := service.ListInboxEvents(context.Background(), query.InboxEventFilter{
		ConversationID:   "27234224",
		ConversationType: "group",
		ObserveOnly:      "true",
		AfterSeq:         0,
		AfterSeqSet:      true,
		Order:            "asc",
		Limit:            10,
	})
	if err != nil {
		t.Fatalf("list inbox events failed: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("expected 1 inbox view, got %d", len(views))
	}
	if !views[0].ObserveOnly {
		t.Fatalf("expected observe-only view")
	}
	if views[0].Content != "B850M board price screenshot" {
		t.Fatalf("unexpected content: %s", views[0].Content)
	}
}
