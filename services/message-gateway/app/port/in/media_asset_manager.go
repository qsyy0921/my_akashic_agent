package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/app/query"
)

type MediaAssetManager interface {
	Register(ctx context.Context, cmd command.RegisterMediaAssetCommand) (query.MediaAssetView, error)
	Get(ctx context.Context, assetID string) (query.MediaAssetView, error)
	List(ctx context.Context, limit int) ([]query.MediaAssetView, error)
}
