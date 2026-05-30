package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type DeliveryAdapterHealthViewer interface {
	CheckDeliveryAdapters(ctx context.Context, filter query.DeliveryAdapterHealthFilter) (query.DeliveryAdapterHealthSummaryView, error)
}
