package model_test

import (
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func TestNewInboxEventValidatesEnvelopeAndDecision(t *testing.T) {
	event, err := model.NewInboxEvent(model.MessageEnvelope{
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
		Content:   "raw group message",
		Timestamp: time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC),
		Metadata:  map[string]string{"observe_only": "true"},
	}, model.LoopDecision{Action: model.LoopActionAllow, Reason: "accepted"}, time.Date(2026, 5, 30, 0, 0, 1, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected inbox event error: %v", err)
	}
	if event.EventID() != "qq:1049511700:group:27234224:1" {
		t.Fatalf("unexpected event id: %s", event.EventID())
	}
	if !event.ObserveOnly() {
		t.Fatalf("expected observe-only metadata to be recognized")
	}
}

func TestInboxEventRejectsMissingDecision(t *testing.T) {
	_, err := model.NewInboxEvent(model.MessageEnvelope{
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
		Content:   "raw group message",
		Timestamp: time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC),
	}, model.LoopDecision{}, time.Date(2026, 5, 30, 0, 0, 1, 0, time.UTC))
	if err == nil {
		t.Fatalf("expected missing decision to fail")
	}
}
