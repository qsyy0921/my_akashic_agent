package auditjsonl_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/auditjsonl"
)

func TestStorePersistsAndReloadsShadowObservedRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shadow-audit.jsonl")
	store, err := auditjsonl.NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	envelope := model.MessageEnvelope{
		EventID: "qq:2365524513:group:27234224:msg-1",
		Channel: model.ChannelRef{
			Kind:             model.ChannelKindQQ,
			AccountID:        "2365524513",
			ConversationID:   "27234224",
			ConversationType: model.ConversationTypeGroup,
		},
		Sender: model.SenderRef{
			ID:   "1049511700",
			Kind: model.SenderKindHuman,
		},
		Content:   "shadow image message",
		Timestamp: time.Date(2026, 5, 30, 0, 40, 0, 0, time.UTC),
		Attachments: []model.Attachment{
			{
				ID:       "asset:qq:image:1",
				Kind:     model.AttachmentKindImage,
				URL:      "file:///E:/agent/akashic/.tmp/image.png",
				MimeType: "image/png",
				Name:     "image.png",
			},
		},
		Metadata: map[string]string{"observe_only": "true"},
		Provenance: model.Provenance{
			Type:        model.ProvenanceHuman,
			ContentHash: "hash-1",
		},
	}

	err = store.RecordMessageDecision(
		context.Background(),
		envelope,
		model.LoopDecision{
			Action: model.LoopActionObserveOnly,
			Reason: "observe-only group",
		},
	)
	if err != nil {
		t.Fatalf("record decision: %v", err)
	}

	reloaded, err := auditjsonl.NewStore(path)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	records, err := reloaded.ListShadowObserved(context.Background(), 10)
	if err != nil {
		t.Fatalf("list observed: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Envelope.EventID != envelope.EventID {
		t.Fatalf("event id changed: %s", records[0].Envelope.EventID)
	}
	if records[0].Envelope.Attachments[0].Name != "image.png" {
		t.Fatalf("attachment metadata not preserved")
	}
	if records[0].Decision.Action != model.LoopActionObserveOnly {
		t.Fatalf("decision action not preserved: %s", records[0].Decision.Action)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read jsonl: %v", err)
	}
	var line map[string]any
	if err := json.Unmarshal(raw[:len(raw)-1], &line); err != nil {
		t.Fatalf("audit json line should be readable json: %v", err)
	}
	if line["event_id"] != envelope.EventID {
		t.Fatalf("jsonl event id missing")
	}
}

func TestStoreReturnsRecentRecordsNewestFirst(t *testing.T) {
	store, err := auditjsonl.NewStore(filepath.Join(t.TempDir(), "shadow-audit.jsonl"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	for _, eventID := range []string{"event-1", "event-2", "event-3"} {
		err := store.RecordMessageDecision(
			context.Background(),
			model.MessageEnvelope{
				EventID: eventID,
				Channel: model.ChannelRef{
					Kind:             model.ChannelKindQQ,
					AccountID:        "2365524513",
					ConversationID:   "27234224",
					ConversationType: model.ConversationTypeGroup,
				},
				Sender:    model.SenderRef{ID: "1049511700", Kind: model.SenderKindHuman},
				Content:   eventID,
				Timestamp: time.Now().UTC(),
			},
			model.LoopDecision{Action: model.LoopActionAllow},
		)
		if err != nil {
			t.Fatalf("record %s: %v", eventID, err)
		}
	}

	records, err := store.ListShadowObserved(context.Background(), 2)
	if err != nil {
		t.Fatalf("list observed: %v", err)
	}

	if got := []string{records[0].Envelope.EventID, records[1].Envelope.EventID}; got[0] != "event-3" || got[1] != "event-2" {
		t.Fatalf("unexpected order: %v", got)
	}
}

