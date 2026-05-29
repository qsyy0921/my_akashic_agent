package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type MediaAssetManager interface {
	Register(ctx context.Context, cmd command.RegisterMediaAssetCommand) (query.MediaAssetView, error)
	Get(ctx context.Context, assetID string) (query.MediaAssetView, error)
	List(ctx context.Context, limit int) ([]query.MediaAssetView, error)
	OpenContent(ctx context.Context, assetID string) (outport.MediaAssetContent, error)
}
