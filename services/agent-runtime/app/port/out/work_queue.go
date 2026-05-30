package outport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type WorkQueuePublisher interface {
	PublishOutboxDelivery(ctx context.Context, delivery model.OutboxDelivery) error
	PublishAgentJob(ctx context.Context, job model.AgentJob) error
}
