package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	domainservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/service"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/memory"
)

func TestMessageIngestServicePublishesEnvelope(t *testing.T) {
	store := memory.NewStore()
	service := newIngestService(store)

	err := service.Ingest(context.Background(), command.IngestMessageCommand{
		EventID: "qq:group:27234224:498",
		Channel: command.ChannelCommand{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   "27234224",
			ConversationType: "group",
		},
		Sender: command.SenderCommand{
			ID:   "2948770636",
			Kind: "human",
		},
		Content:   "hello",
		Timestamp: time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("ingest returned error: %v", err)
	}

	if len(store.Observed()) != 1 {
		t.Fatalf("expected 1 observed event, got %d", len(store.Observed()))
	}
	if len(store.AgentInbound()) != 1 {
		t.Fatalf("expected 1 agent inbound event, got %d", len(store.AgentInbound()))
	}
	if store.AgentInbound()[0].Channel.AccountID != "1049511700" {
		t.Fatalf("unexpected account id: %s", store.AgentInbound()[0].Channel.AccountID)
	}
}

func TestMessageIngestServiceObservesSelfEchoWithoutAgentInbound(t *testing.T) {
	store := memory.NewStore()
	service := newIngestService(store)

	err := service.Ingest(context.Background(), command.IngestMessageCommand{
		EventID: "qq:private:2365524513:1",
		Channel: command.ChannelCommand{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   "2365524513",
			ConversationType: "private",
		},
		Sender: command.SenderCommand{
			ID:   "2365524513",
			Kind: string(model.SenderKindBot),
		},
		Content:   "plain bot message",
		Timestamp: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("ingest returned error: %v", err)
	}

	observed := store.Observed()
	if len(observed) != 1 {
		t.Fatalf("expected 1 observed event, got %d", len(observed))
	}
	if observed[0].Decision.Action != model.LoopActionObserveOnly {
		t.Fatalf("expected observe-only decision, got %s", observed[0].Decision.Action)
	}
	if len(store.AgentInbound()) != 0 {
		t.Fatalf("expected no agent inbound events, got %d", len(store.AgentInbound()))
	}
}

func TestMessageIngestServiceShadowIngestDoesNotPublishAgentInbound(t *testing.T) {
	store := memory.NewStore()
	service := newIngestService(store)

	decision, err := service.ShadowIngest(context.Background(), command.IngestMessageCommand{
		EventID: "qq:private:1049511700:2",
		Channel: command.ChannelCommand{
			Kind:             "qq",
			AccountID:        "2365524513",
			ConversationID:   "1049511700",
			ConversationType: "private",
		},
		Sender: command.SenderCommand{
			ID:   "2948770636",
			Kind: string(model.SenderKindHuman),
		},
		Content:   "/ask hello",
		Timestamp: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("shadow ingest returned error: %v", err)
	}
	if decision.Action != model.LoopActionAllow {
		t.Fatalf("expected allow decision, got %s", decision.Action)
	}
	if len(store.Observed()) != 1 {
		t.Fatalf("expected 1 observed event, got %d", len(store.Observed()))
	}
	if len(store.Audits()) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(store.Audits()))
	}
	if len(store.AgentInbound()) != 0 {
		t.Fatalf("shadow ingest must not publish agent inbound, got %d", len(store.AgentInbound()))
	}
}

func TestMessageIngestServiceShadowIngestRegistersAttachments(t *testing.T) {
	store := memory.NewStore()
	service := newIngestService(store)

	_, err := service.ShadowIngest(context.Background(), command.IngestMessageCommand{
		EventID: "qq:1049511700:group:27234224:msg-498",
		Channel: command.ChannelCommand{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   "27234224",
			ConversationType: "group",
		},
		Sender: command.SenderCommand{
			ID:   "2948770636",
			Kind: string(model.SenderKindHuman),
		},
		Content: "/image evidence",
		Attachments: []command.AttachmentCommand{{
			ID:        "asset:qq:image:27234224:msg-498:1",
			Kind:      "image",
			URL:       "E:/agent/akashic/.akashic-workspace/uploads/qq-image.png",
			MimeType:  "image/png",
			Name:      "qq-image.png",
			SizeBytes: 123,
		}},
		Timestamp: time.Now().UTC(),
		Metadata: map[string]string{
			"shadow_mode":  "true",
			"observe_only": "true",
			"session_key":  "qq:gqq:27234224",
		},
	})
	if err != nil {
		t.Fatalf("shadow ingest returned error: %v", err)
	}

	assets := store.MediaAssets()
	if len(assets) != 1 {
		t.Fatalf("expected 1 media asset, got %d", len(assets))
	}
	asset := assets[0]
	if asset.AssetID != "asset:qq:image:27234224:msg-498:1" {
		t.Fatalf("unexpected asset id: %s", asset.AssetID)
	}
	if asset.SourceMessageID != "qq:1049511700:group:27234224:msg-498" {
		t.Fatalf("unexpected source message id: %s", asset.SourceMessageID)
	}
	if asset.Channel.AccountID != "1049511700" {
		t.Fatalf("unexpected account id: %s", asset.Channel.AccountID)
	}
	if asset.Metadata["registered_from"] != "message_ingest" {
		t.Fatalf("missing registration metadata: %+v", asset.Metadata)
	}
}

func TestMessageIngestServiceShadowIngestRecordsInboxEvent(t *testing.T) {
	store := memory.NewStore()
	service := appservice.NewMessageIngestServiceWithRuntimeStores(
		store,
		store,
		store,
		store,
		domainservice.NewProvenanceClassifier([]string{"1049511700", "2365524513"}),
		domainservice.NewLoopGuard([]string{"1049511700", "2365524513"}, 15*time.Second, 6),
		store,
		store,
	)

	_, err := service.ShadowIngest(context.Background(), command.IngestMessageCommand{
		EventID: "qq:1049511700:group:27234224:msg-raw-1",
		Channel: command.ChannelCommand{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   "27234224",
			ConversationType: "group",
		},
		Sender: command.SenderCommand{
			ID:   "2948770636",
			Kind: string(model.SenderKindHuman),
		},
		Content:   "raw group observation",
		Timestamp: time.Now().UTC(),
		Metadata:  map[string]string{"observe_only": "true"},
	})
	if err != nil {
		t.Fatalf("shadow ingest returned error: %v", err)
	}

	events := store.InboxEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 inbox event, got %d", len(events))
	}
	if events[0].Envelope.EventID != "qq:1049511700:group:27234224:msg-raw-1" {
		t.Fatalf("unexpected inbox event id: %s", events[0].Envelope.EventID)
	}
	if !events[0].ObserveOnly() {
		t.Fatalf("expected inbox event to preserve observe-only state")
	}
}

func newIngestService(store *memory.Store) *appservice.MessageIngestService {
	botIDs := []string{"1049511700", "2365524513"}
	return appservice.NewMessageIngestServiceWithRuntimeStores(
		store,
		store,
		store,
		store,
		domainservice.NewProvenanceClassifier(botIDs),
		domainservice.NewLoopGuard(botIDs, 15*time.Second, 6),
		store,
		store,
	)
}
