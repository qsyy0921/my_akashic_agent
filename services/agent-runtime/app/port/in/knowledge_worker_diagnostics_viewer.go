package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type KnowledgeWorkerDiagnosticsViewer interface {
	Get(ctx context.Context, filter query.KnowledgeWorkerDiagnosticsFilter) (query.KnowledgeWorkerDiagnosticsView, error)
}
