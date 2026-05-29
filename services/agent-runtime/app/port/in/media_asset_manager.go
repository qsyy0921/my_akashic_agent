package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type MediaAssetManager interface {
	Register(ctx context.Context, cmd command.RegisterMediaAssetCommand) (query.MediaAssetView, error)
	Get(ctx context.Context, assetID string) (query.MediaAssetView, error)
	List(ctx context.Context, limit int) ([]query.MediaAssetView, error)
}

