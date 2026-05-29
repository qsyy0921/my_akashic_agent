package mediaassetstore_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	store "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/mediaassetstore"
)

func TestMediaAssetStorePersistsAssetsAcrossRestarts(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "media-assets.json")
	repo, err := store.NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	now := time.Date(2026, 5, 30, 7, 30, 0, 0, time.UTC)
	assetA := newAsset(t, "asset:a", "a.png", now)
	assetB := newAsset(t, "asset:b", "b.txt", now.Add(time.Minute))
	if err := repo.SaveMediaAsset(ctx, assetA); err != nil {
		t.Fatalf("save assetA: %v", err)
	}
	if err := repo.SaveMediaAsset(ctx, assetB); err != nil {
		t.Fatalf("save assetB: %v", err)
	}

	reloaded, err := store.NewStore(path)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	stored, ok, err := reloaded.FindMediaAsset(ctx, "asset:a")
	if err != nil {
		t.Fatalf("find stored asset: %v", err)
	}
	if !ok {
		t.Fatalf("expected asset:a to persist")
	}
	if stored.Name != "a.png" {
		t.Fatalf("expected name to persist, got %s", stored.Name)
	}
	items, err := reloaded.ListMediaAssets(ctx, query.MediaAssetFilter{Limit: 10})
	if err != nil {
		t.Fatalf("list assets: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 assets, got %d", len(items))
	}
	if items[0].AssetID != "asset:b" {
		t.Fatalf("expected newest insertion first, got %+v", items)
	}
	filtered, err := reloaded.ListMediaAssets(ctx, query.MediaAssetFilter{
		Limit:                 10,
		ChannelKind:           "qq",
		ConversationID:        "27234224",
		ConversationType:      "group",
		SourceMessageIDSuffix: "498",
	})
	if err != nil {
		t.Fatalf("list filtered assets: %v", err)
	}
	if len(filtered) != 2 {
		t.Fatalf("expected both persisted assets to match filters, got %+v", filtered)
	}
}

func TestMediaAssetStorePersistsUpdatedAsset(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "media-assets-update.json")
	repo, err := store.NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	now := time.Date(2026, 5, 30, 8, 0, 0, 0, time.UTC)
	asset := newAsset(t, "asset:update", "before.png", now)
	if err := repo.SaveMediaAsset(ctx, asset); err != nil {
		t.Fatalf("save asset: %v", err)
	}
	asset.Name = "after.png"
	asset.UpdatedAt = now.Add(time.Minute)
	if err := repo.SaveMediaAsset(ctx, asset); err != nil {
		t.Fatalf("save updated asset: %v", err)
	}

	reloaded, err := store.NewStore(path)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	stored, ok, err := reloaded.FindMediaAsset(ctx, "asset:update")
	if err != nil {
		t.Fatalf("find stored: %v", err)
	}
	if !ok {
		t.Fatalf("expected stored asset")
	}
	if stored.Name != "after.png" {
		t.Fatalf("expected updated name, got %s", stored.Name)
	}
	items, err := reloaded.ListMediaAssets(ctx, query.MediaAssetFilter{Limit: 10})
	if err != nil {
		t.Fatalf("list assets: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one idempotent asset, got %d", len(items))
	}
}

func newAsset(t *testing.T, assetID string, name string, now time.Time) model.MediaAsset {
	t.Helper()
	asset, err := model.NewMediaAsset(model.ChannelRef{
		Kind:             "qq",
		AccountID:        "1049511700",
		ConversationID:   "27234224",
		ConversationType: "group",
	}, model.MediaAssetSpec{
		AssetID:         assetID,
		SourceMessageID: "qq:gqq:27234224:498",
		SenderID:        "2948770636",
		Kind:            model.MediaAssetImage,
		URL:             "E:/agent/akashic/.akashic-workspace/uploads/" + name,
		MimeType:        "image/png",
		Name:            name,
		SizeBytes:       123,
		Metadata:        map[string]string{"registered_from": "test"},
	}, now)
	if err != nil {
		t.Fatalf("new asset: %v", err)
	}
	return asset
}
