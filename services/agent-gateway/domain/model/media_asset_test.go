package model_test

import (
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/domain/model"
)

func TestNewMediaAssetDefaultsRetentionAndValidatesRoute(t *testing.T) {
	now := time.Date(2026, 5, 30, 4, 30, 0, 0, time.UTC)
	asset, err := model.NewMediaAsset(model.ChannelRef{
		Kind:             "qq",
		AccountID:        "1049511700",
		ConversationID:   "27234224",
		ConversationType: "group",
	}, model.MediaAssetSpec{
		AssetID:         "asset:qq:1049511700:group:27234224:msg-1:1",
		SourceMessageID: "msg-1",
		SenderID:        "2948770636",
		Kind:            model.MediaAssetImage,
		URL:             "https://example.invalid/image.png",
		MimeType:        "image/png",
		Name:            "image.png",
	}, now)
	if err != nil {
		t.Fatalf("new media asset: %v", err)
	}
	if asset.Retention != "default" {
		t.Fatalf("expected default retention, got %s", asset.Retention)
	}
	if asset.Channel.AccountID != "1049511700" {
		t.Fatalf("expected account id to be preserved, got %s", asset.Channel.AccountID)
	}
}

func TestMediaAssetRejectsMissingSourceMessage(t *testing.T) {
	_, err := model.NewMediaAsset(model.ChannelRef{
		Kind:             "qq",
		AccountID:        "1049511700",
		ConversationID:   "27234224",
		ConversationType: "group",
	}, model.MediaAssetSpec{
		AssetID:  "asset:1",
		SenderID: "2948770636",
		Kind:     model.MediaAssetFile,
		Name:     "guide.pdf",
	}, time.Now().UTC())
	if err == nil {
		t.Fatal("expected missing source message id to fail")
	}
}
