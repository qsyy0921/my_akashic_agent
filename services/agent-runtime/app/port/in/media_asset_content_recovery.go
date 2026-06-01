package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type MediaAssetContentRecoveryPreflightChecker interface {
	CheckMediaAssetContentRecoveryPreflight(ctx context.Context, filter query.MediaAssetContentRecoveryPreflightFilter) (query.MediaAssetContentRecoveryPreflightView, error)
}

type MediaAssetContentRecoverer interface {
	RecoverMediaAssetContent(ctx context.Context, cmd command.RecoverMediaAssetContentCommand) (query.MediaAssetContentRecoveryView, error)
}
