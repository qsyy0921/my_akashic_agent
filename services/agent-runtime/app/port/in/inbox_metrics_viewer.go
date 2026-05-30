package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type InboxMetricsViewer interface {
	Get(ctx context.Context, filter query.InboxMetricsFilter) (query.InboxMetricsView, error)
}
