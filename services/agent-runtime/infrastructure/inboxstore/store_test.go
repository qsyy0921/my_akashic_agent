package inboxstore_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/inboxstore"
)

func TestInboxStorePersistsAndFiltersEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inbox.json")
	store, err := inboxstore.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	event := newInboxEvent(t, "qq:1049511700:group:27234224:1", "27234224", "raw hardware note")
	if err := store.SaveInboxEvent(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveInboxEvent(context.Background(), event); err != nil {
		t.Fatal(err)
	}

	reopened, err := inboxstore.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	items, err := reopened.ListInboxEvents(context.Background(), query.InboxEventFilter{
		ConversationID:   "27234224",
		ConversationType: "group",
		ObserveOnly:      "true",
		Limit:            10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected idempotent persisted inbox event, got %d", len(items))
	}
	if items[0].Envelope.Content != "raw hardware note" {
		t.Fatalf("unexpected content: %s", items[0].Envelope.Content)
	}
}

func TestInboxStoreFindsEvent(t *testing.T) {
	store, err := inboxstore.NewStore(filepath.Join(t.TempDir(), "inbox.json"))
	if err != nil {
		t.Fatal(err)
	}
	event := newInboxEvent(t, "qq:1049511700:group:187890369:2", "187890369", "game group note")
	if err := store.SaveInboxEvent(context.Background(), event); err != nil {
		t.Fatal(err)
	}

	found, ok, err := store.FindInboxEvent(context.Background(), "qq:1049511700:group:187890369:2")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("expected event to be found")
	}
	if found.Envelope.Channel.ConversationID != "187890369" {
		t.Fatalf("unexpected group id: %s", found.Envelope.Channel.ConversationID)
	}
}

func newInboxEvent(t *testing.T, eventID string, groupID string, content string) model.InboxEvent {
	t.Helper()
	event, err := model.NewInboxEvent(model.MessageEnvelope{
		EventID: eventID,
		Channel: model.ChannelRef{
			Kind:             model.ChannelKindQQ,
			AccountID:        "1049511700",
			ConversationID:   groupID,
			ConversationType: model.ConversationTypeGroup,
		},
		Sender: model.SenderRef{
			ID:   "2948770636",
			Kind: model.SenderKindHuman,
		},
		Content:   content,
		Timestamp: time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC),
		Metadata:  map[string]string{"observe_only": "true"},
	}, model.LoopDecision{Action: model.LoopActionAllow, Reason: "accepted"}, time.Date(2026, 5, 30, 0, 0, 1, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return event
}
