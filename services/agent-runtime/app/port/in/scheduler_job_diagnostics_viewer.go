package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type SchedulerJobDiagnosticsViewer interface {
	GetSchedulerJobDiagnostics(ctx context.Context, filter query.SchedulerJobDiagnosticsFilter) (query.SchedulerJobDiagnosticsView, error)
}
