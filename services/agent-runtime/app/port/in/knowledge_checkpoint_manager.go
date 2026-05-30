package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type KnowledgeCheckpointManager interface {
	List(ctx context.Context, filter query.KnowledgeCheckpointFilter) ([]query.KnowledgeCheckpointView, error)
	Get(ctx context.Context, checkpointID string) (query.KnowledgeCheckpointView, error)
	Upsert(ctx context.Context, cmd command.UpsertKnowledgeCheckpointCommand) (query.KnowledgeCheckpointView, error)
}
