package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type KnowledgeCheckpointManager interface {
	Get(ctx context.Context, checkpointID string) (query.KnowledgeCheckpointView, error)
	Upsert(ctx context.Context, cmd command.UpsertKnowledgeCheckpointCommand) (query.KnowledgeCheckpointView, error)
}
