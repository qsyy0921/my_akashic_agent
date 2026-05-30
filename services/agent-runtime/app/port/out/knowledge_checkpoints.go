package outport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type KnowledgeCheckpointRepository interface {
	SaveKnowledgeCheckpoint(ctx context.Context, checkpoint model.KnowledgeCheckpoint) error
	FindKnowledgeCheckpoint(ctx context.Context, checkpointID string) (model.KnowledgeCheckpoint, bool, error)
}
