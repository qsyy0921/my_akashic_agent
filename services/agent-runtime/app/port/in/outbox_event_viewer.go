package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type OutboxDeliveryEventViewer interface {
	List(ctx context.Context, filter query.OutboxDeliveryEventFilter) ([]query.OutboxDeliveryEventView, error)
}
