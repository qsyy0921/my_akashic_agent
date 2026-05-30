package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type QueueBackendViewer interface {
	Get(ctx context.Context) (query.QueueBackendView, error)
}
