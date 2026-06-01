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
	List(ctx context.Context, filter query.MediaAssetFilter) ([]query.MediaAssetView, error)
	OpenContent(ctx context.Context, assetID string) (outport.MediaAssetContent, error)
	ContentAccessPlan(ctx context.Context, assetID string) (query.MediaAssetContentAccessPlanView, error)
	ContentRecoveryPlan(ctx context.Context, assetID string) (query.MediaAssetContentRecoveryPlanView, error)
	ContentDiagnostics(ctx context.Context, filter query.MediaAssetContentDiagnosticsFilter) (query.MediaAssetContentDiagnosticsView, error)
	RetentionDiagnostics(ctx context.Context, filter query.MediaAssetRetentionDiagnosticsFilter) (query.MediaAssetRetentionDiagnosticsView, error)
	RetentionPlan(ctx context.Context, filter query.MediaAssetRetentionDiagnosticsFilter) (query.MediaAssetRetentionPlanView, error)
}
