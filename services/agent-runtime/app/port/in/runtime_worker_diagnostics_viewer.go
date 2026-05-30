package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type RuntimeWorkerDiagnosticsViewer interface {
	GetRuntimeWorkers(ctx context.Context) (query.RuntimeWorkerDiagnosticsView, error)
}
