package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type QueueTopologyViewer interface {
	GetQueueTopology(ctx context.Context) (query.QueueTopologyView, error)
}
