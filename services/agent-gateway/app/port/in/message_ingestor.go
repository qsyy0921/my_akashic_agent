package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/app/command"
)

type MessageIngestor interface {
	Ingest(ctx context.Context, cmd command.IngestMessageCommand) error
}
