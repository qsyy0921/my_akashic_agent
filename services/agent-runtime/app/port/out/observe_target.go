package outport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type ObserveTargetRepository interface {
	SaveObserveTargets(ctx context.Context, targets []model.ObserveTarget) error
	ListObserveTargets(ctx context.Context) ([]model.ObserveTarget, error)
}
