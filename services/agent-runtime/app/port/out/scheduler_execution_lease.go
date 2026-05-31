package outport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type SchedulerExecutionLeaseRepository interface {
	SaveSchedulerExecutionLease(ctx context.Context, lease model.SchedulerExecutionLease) error
	DeleteSchedulerExecutionLease(ctx context.Context, jobID string) error
	ListSchedulerExecutionLeases(ctx context.Context) ([]model.SchedulerExecutionLease, error)
}
