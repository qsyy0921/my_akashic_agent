package outport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type DeliveryAdapter interface {
	SupportsDeliveryChannel(channel string) bool
	DispatchDeliveryStep(ctx context.Context, step model.DeliveryDispatchStep) (model.DeliveryDispatchResult, error)
}
