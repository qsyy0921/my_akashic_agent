package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type ObserveCaptureDiagnosticsViewer interface {
	GetObserveCaptureDiagnostics(ctx context.Context, filter query.ObserveCaptureDiagnosticsFilter) (query.ObserveCaptureDiagnosticsView, error)
}
