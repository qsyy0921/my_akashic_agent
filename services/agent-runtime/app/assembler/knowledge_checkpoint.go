package assembler

import (
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func ToKnowledgeCheckpointView(checkpoint model.KnowledgeCheckpoint) query.KnowledgeCheckpointView {
	return query.KnowledgeCheckpointView{
		CheckpointID: checkpoint.CheckpointID,
		Cursor:       checkpoint.Cursor,
		UpdatedAt:    formatKnowledgeCheckpointTime(checkpoint.UpdatedAt),
		Metadata:     checkpoint.Metadata,
	}
}

func formatKnowledgeCheckpointTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
