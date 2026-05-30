package outport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type DeliveryAdapterHealthProbe interface {
	CheckDeliveryAdapterHealth(ctx context.Context, filter query.DeliveryAdapterHealthFilter) ([]query.DeliveryAdapterHealthView, error)
}
