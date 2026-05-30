package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type OutboxMetricsViewer interface {
	Get(ctx context.Context, filter query.OutboxMetricsFilter) (query.OutboxMetricsView, error)
}
