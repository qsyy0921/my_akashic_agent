package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/app/command"
	appservice "github.com/kachofugetsu09/akashic-agent/services/message-gateway/app/service"
	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/infrastructure/memory"
)

func TestMediaAssetServiceRegistersGeneratedIDAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := appservice.NewMediaAssetService(store)
	now := time.Date(2026, 5, 30, 5, 0, 0, 0, time.UTC)

	cmd := command.RegisterMediaAssetCommand{
		Channel: command.ChannelCommand{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   "27234224",
			ConversationType: "group",
		},
		SourceMessageID: "qq:gqq:27234224:498",
		SenderID:        "2948770636",
		Kind:            "image",
		URL:             "https://example.invalid/image.png",
		MimeType:        "image/png",
		Name:            "image.png",
		Index:           2,
		Timestamp:       now,
	}
	first, err := service.Register(ctx, cmd)
	if err != nil {
		t.Fatalf("register media asset: %v", err)
	}
	second, err := service.Register(ctx, cmd)
	if err != nil {
		t.Fatalf("register media asset again: %v", err)
	}
	if second.AssetID != first.AssetID {
		t.Fatalf("expected idempotent asset id, got %s and %s", first.AssetID, second.AssetID)
	}
	if len(store.MediaAssets()) != 1 {
		t.Fatalf("expected one stored asset, got %d", len(store.MediaAssets()))
	}
	if first.AssetID != "asset:qq:1049511700:group:27234224:qq:gqq:27234224:498:2" {
		t.Fatalf("unexpected generated asset id: %s", first.AssetID)
	}
}

func TestMediaAssetServiceListsNewestFirst(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := appservice.NewMediaAssetService(store)
	now := time.Date(2026, 5, 30, 5, 0, 0, 0, time.UTC)

	for _, item := range []struct {
		id   string
		name string
	}{
		{id: "asset:old", name: "old.png"},
		{id: "asset:new", name: "new.png"},
	} {
		_, err := service.Register(ctx, command.RegisterMediaAssetCommand{
			AssetID: item.id,
			Channel: command.ChannelCommand{
				Kind:             "qq",
				AccountID:        "1049511700",
				ConversationID:   "27234224",
				ConversationType: "group",
			},
			SourceMessageID: item.id + ":msg",
			SenderID:        "2948770636",
			Kind:            "image",
			Name:            item.name,
			Timestamp:       now,
		})
		if err != nil {
			t.Fatalf("register %s: %v", item.id, err)
		}
	}

	items, err := service.List(ctx, 10)
	if err != nil {
		t.Fatalf("list media assets: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 assets, got %d", len(items))
	}
	if items[0].AssetID != "asset:new" {
		t.Fatalf("expected newest first, got %+v", items)
	}
}
