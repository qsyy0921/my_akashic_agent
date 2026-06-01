package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type MediaAssetRetentionCleanupPreflightChecker interface {
	CheckMediaAssetRetentionCleanupPreflight(ctx context.Context, filter query.MediaAssetRetentionCleanupPreflightFilter) (query.MediaAssetRetentionCleanupPreflightView, error)
}
