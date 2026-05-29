package outport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/domain/model"
)

type MediaAssetRepository interface {
	SaveMediaAsset(ctx context.Context, asset model.MediaAsset) error
	FindMediaAsset(ctx context.Context, assetID string) (model.MediaAsset, bool, error)
	ListMediaAssets(ctx context.Context, limit int) ([]model.MediaAsset, error)
}
