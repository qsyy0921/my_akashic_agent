package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type KnowledgePipelineDiagnosticsViewer interface {
	GetKnowledgePipelineDiagnostics(ctx context.Context, filter query.KnowledgePipelineDiagnosticsFilter) (query.KnowledgePipelineDiagnosticsView, error)
}
