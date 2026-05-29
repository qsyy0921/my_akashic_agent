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

func newIngestService(store *memory.Store) *appservice.MessageIngestService {
	botIDs := []string{"1049511700", "2365524513"}
	return appservice.NewMessageIngestService(
		store,
		store,
		store,
		store,
		domainservice.NewProvenanceClassifier(botIDs),
		domainservice.NewLoopGuard(botIDs, 15*time.Second, 6),
	)
}

