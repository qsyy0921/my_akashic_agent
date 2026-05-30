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

func TestInboxMetricsServiceSummarizesObserveOnlyCollection(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 5, 30, 1, 0, 0, 0, time.UTC)

	for _, event := range []model.InboxEvent{
		mustInboxEvent(t, "qq:1049511700:group:27234224:1", model.ChannelKindQQ, "1049511700", "27234224", model.ConversationTypeGroup, "2948770636", model.SenderKindHuman, "hardware note", nil, map[string]string{"observe_only": "true", "seq": "1"}, model.LoopActionAllow, now),
		mustInboxEvent(t, "qq:1049511700:group:27234224:2", model.ChannelKindQQ, "1049511700", "27234224", model.ConversationTypeGroup, "99887766", model.SenderKindHuman, "board image", []model.Attachment{{
			ID:       "asset:qq:image:1",
			Kind:     model.AttachmentKindImage,
			MimeType: "image/png",
			Name:     "board.png",
		}}, map[string]string{"observe_only": "true", "seq": "2"}, model.LoopActionAllow, now.Add(time.Second)),
		mustInboxEvent(t, "telegram:bot:private:123:1", model.ChannelKindTelegram, "bot", "123", model.ConversationTypePrivate, "123", model.SenderKindHuman, "/ask hello", nil, map[string]string{"seq": "7"}, model.LoopActionAllow, now.Add(2*time.Second)),
	} {
		if err := store.SaveInboxEvent(ctx, event); err != nil {
			t.Fatalf("save inbox event: %v", err)
		}
	}

	metrics := appservice.NewInboxMetricsService(store)
	view, err := metrics.Get(ctx, query.InboxMetricsFilter{Limit: 10})
	if err != nil {
		t.Fatalf("get metrics: %v", err)
	}
	if view.SampledEvents != 3 || view.ObserveOnlyTotal != 2 || view.ReplyEligibleTotal != 1 {
		t.Fatalf("unexpected top-level metrics: %+v", view)
	}
	if view.WithAttachments != 1 || view.AttachmentCount != 1 || view.UniqueSenders != 3 {
		t.Fatalf("unexpected attachment/sender metrics: %+v", view)
	}
	if view.EventsByChannelKind["qq"].ObserveOnly != 2 ||
		view.EventsByChannelKind["telegram"].ReplyEligible != 1 {
		t.Fatalf("unexpected channel metrics: %+v", view.EventsByChannelKind)
	}
	groupKey := "qq/1049511700/group/27234224"
	group := view.EventsByConversation[groupKey]
	if group.Total != 2 || group.UniqueSenders != 2 || group.LatestSeq != 2 || group.SequencedEvents != 2 {
		t.Fatalf("unexpected group conversation metrics: %+v", group)
	}
	if len(view.Recent) != 3 || view.Recent[0].EventID != "telegram:bot:private:123:1" {
		t.Fatalf("expected newest-first recent samples, got %+v", view.Recent)
	}

	qqView, err := metrics.Get(ctx, query.InboxMetricsFilter{
		Limit:            10,
		ChannelKind:      "qq",
		ConversationID:   "27234224",
		ConversationType: "group",
		ObserveOnly:      "true",
	})
	if err != nil {
		t.Fatalf("get filtered metrics: %v", err)
	}
	if qqView.SampledEvents != 2 || qqView.ReplyEligibleTotal != 0 {
		t.Fatalf("unexpected filtered metrics: %+v", qqView)
	}
}

func mustInboxEvent(
	t *testing.T,
	eventID string,
	channelKind model.ChannelKind,
	accountID string,
	conversationID string,
	conversationType model.ConversationType,
	senderID string,
	senderKind model.SenderKind,
	content string,
	attachments []model.Attachment,
	metadata map[string]string,
	action model.LoopAction,
	timestamp time.Time,
) model.InboxEvent {
	t.Helper()
	event, err := model.NewInboxEvent(model.MessageEnvelope{
		EventID: eventID,
		Channel: model.ChannelRef{
			Kind:             channelKind,
			AccountID:        accountID,
			ConversationID:   conversationID,
			ConversationType: conversationType,
		},
		Sender: model.SenderRef{
			ID:   senderID,
			Kind: senderKind,
		},
		Content:     content,
		Attachments: attachments,
		Timestamp:   timestamp,
		Metadata:    metadata,
	}, model.LoopDecision{Action: action, Reason: "fixture"}, timestamp)
	if err != nil {
		t.Fatalf("new inbox event: %v", err)
	}
	return event
}
