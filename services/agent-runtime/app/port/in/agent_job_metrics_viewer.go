package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type AgentJobMetricsViewer interface {
	Get(ctx context.Context, filter query.AgentJobMetricsFilter) (query.AgentJobMetricsView, error)
}
