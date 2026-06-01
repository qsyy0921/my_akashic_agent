package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type MediaAssetRetentionCleaner interface {
	CleanupMediaAssetRetention(ctx context.Context, cmd command.CleanupMediaAssetRetentionCommand) (query.MediaAssetRetentionCleanupView, error)
}
