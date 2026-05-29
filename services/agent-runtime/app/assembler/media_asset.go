package assembler

import (
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func ToMediaAssetView(asset model.MediaAsset) query.MediaAssetView {
	return query.MediaAssetView{
		AssetID: asset.AssetID,
		Channel: query.MediaAssetChannelView{
			Kind:             string(asset.Channel.Kind),
			AccountID:        asset.Channel.AccountID,
			ConversationID:   asset.Channel.ConversationID,
			ConversationType: string(asset.Channel.ConversationType),
		},
		SourceMessageID: asset.SourceMessageID,
		SenderID:        asset.SenderID,
		Kind:            string(asset.Kind),
		URL:             asset.URL,
		MimeType:        asset.MimeType,
		Name:            asset.Name,
		SizeBytes:       asset.SizeBytes,
		ContentHash:     asset.ContentHash,
		Retention:       asset.Retention,
		CreatedAt:       formatMediaTime(asset.CreatedAt),
		UpdatedAt:       formatMediaTime(asset.UpdatedAt),
		Metadata:        asset.Metadata,
	}
}

func ToMediaAssetViews(items []model.MediaAsset) []query.MediaAssetView {
	views := make([]query.MediaAssetView, 0, len(items))
	for _, item := range items {
		views = append(views, ToMediaAssetView(item))
	}
	return views
}

func formatMediaTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

