package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type DeliveryAdapterDiagnosticsViewer interface {
	ListDeliveryAdapters(ctx context.Context) ([]query.DeliveryAdapterDiagnosticsView, error)
}
