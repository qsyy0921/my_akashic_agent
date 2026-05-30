package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type RuntimeOverviewViewer interface {
	Get(ctx context.Context, filter query.RuntimeOverviewFilter) (query.RuntimeOverviewView, error)
}
