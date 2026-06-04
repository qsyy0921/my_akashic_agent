package outport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type AgentWorkerStatusRepository interface {
	SaveAgentWorkerStatus(ctx context.Context, status model.AgentWorkerStatus) error
	DeleteAgentWorkerStatus(ctx context.Context, workerID string) error
	ListAgentWorkerStatuses(ctx context.Context) ([]model.AgentWorkerStatus, error)
}
