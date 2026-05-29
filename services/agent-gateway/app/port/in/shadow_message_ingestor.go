package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/domain/model"
)

type ShadowMessageIngestor interface {
	ShadowIngest(ctx context.Context, cmd command.IngestMessageCommand) (model.LoopDecision, error)
}
